package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"go/format"
	options "google.golang.org/genproto/googleapis/api/annotations"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var restTemplate = template.Must(template.New("rest").Parse(`
package {{.Pkg}}

import (
	"github.com/gorilla/mux"
	"github.com/tfeng/postgres-grpc-example/rest"
	"golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	"net/http"
)

var (
	_ = rest.HandleRequest
	_ http.Server
)

{{range $svc := .Services}}
func Create{{$svc.Service.GetName}}Router(ctx context.Context, impl {{$svc.Service.GetName}}Server, interceptor grpc.UnaryServerInterceptor, s *grpc.Server) (*mux.Router, error) {
	r := mux.NewRouter()
	{{range $m := $svc.Methods}}
	{{range $o := $m.HttpOpts}}
	r.HandleFunc({{$o.PathTemplate | printf "%q"}}, func(w http.ResponseWriter, r *http.Request) {
		rest.HandleRequest(ctx, interceptor, s, w, r, &{{$m.InputType}}{}, func(ctx context.Context, req interface{}) (interface{}, error) {
			return impl.{{$m.Method.GetName}}(ctx, req.(*{{$m.InputType}}))
		})
	}).Methods({{$o.HttpMethod | printf "%q"}}).Headers("content-type", "application/json")

	r.HandleFunc({{$o.PathTemplate | printf "%q"}}, func(w http.ResponseWriter, r *http.Request) {
		rest.HandleWrongContentType(ctx, w, r)
	}).Methods({{$o.HttpMethod | printf "%q"}})
	{{end}}
	{{end}}
	return r, nil
}
{{end}}
`))

type HttpOpt struct {
	HttpMethod   string
	PathTemplate string
}

type Method struct {
	Method     *descriptor.MethodDescriptorProto
	InputType  string
	OutputType string
	HttpOpts   []*HttpOpt
}

type Service struct {
	Service *descriptor.ServiceDescriptorProto
	Methods []Method
}

type TemplateData struct {
	Pkg      string
	Services []Service
}

func createTemplateData(file *descriptor.FileDescriptorProto) TemplateData {
	var services []Service
	for _, svc := range file.GetService() {
		var methods []Method
		for _, md := range svc.GetMethod() {
			inputType := md.GetInputType()
			inputComponents := strings.Split(inputType, ".")
			simpleInputType := inputComponents[len(inputComponents)-1]

			outputType := md.GetOutputType()
			outputComponents := strings.Split(outputType, ".")
			simpleOutputType := outputComponents[len(outputComponents)-1]
			methods = append(methods, Method{
				md,
				simpleInputType,
				simpleOutputType,
				getHttpOpts(md)})
		}
		services = append(services, Service{svc, methods})
	}
	return TemplateData{
		*file.Package,
		services,
	}
}

func extractHttpRule(opts *options.HttpRule) (string, string, error) {
	var httpMethod, pathTemplate string
	switch {
	case opts.GetGet() != "":
		httpMethod = "GET"
		pathTemplate = opts.GetGet()
		if opts.Body != "" {
			return "", "", fmt.Errorf("needs request body even though http method is GET: %s", opts.Body)
		}

	case opts.GetPut() != "":
		httpMethod = "PUT"
		pathTemplate = opts.GetPut()

	case opts.GetPost() != "":
		httpMethod = "POST"
		pathTemplate = opts.GetPost()

	case opts.GetDelete() != "":
		httpMethod = "DELETE"
		pathTemplate = opts.GetDelete()

	case opts.GetPatch() != "":
		httpMethod = "PATCH"
		pathTemplate = opts.GetPatch()

	case opts.GetCustom() != nil:
		custom := opts.GetCustom()
		httpMethod = custom.Kind
		pathTemplate = custom.Path

	default:
		glog.V(1).Infof("no pattern specified in google.api.HttpRule: %s", opts.Body)
		return "", "", nil
	}
	return httpMethod, pathTemplate, nil
}

func getHttpOpts(md *descriptor.MethodDescriptorProto) []*HttpOpt {
	var httpOpts []*HttpOpt
	if proto.HasExtension(md.Options, options.E_Http) {
		if ext, err := proto.GetExtension(md.Options, options.E_Http); err != nil {
			glog.Fatal("unable to get http options", err)
		} else {
			opts := ext.(*options.HttpRule)
			if httpMethod, pathTemplate, err := extractHttpRule(opts); err != nil {
				glog.Fatal("unable to process http rule", err)
			} else {
				httpOpts = append(httpOpts, &HttpOpt{httpMethod, pathTemplate})
			}
			for _, addOpts := range opts.AdditionalBindings {
				if httpMethod, pathTemplate, err := extractHttpRule(addOpts); err != nil {
					glog.Fatal("unable to process http rule", err)
				} else {
					httpOpts = append(httpOpts, &HttpOpt{httpMethod, pathTemplate})
				}
			}
		}
	}
	return httpOpts
}

func main() {
	flag.Parse()
	gen := generator.New()

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		glog.Fatal("unable to read input", err)
	}

	if err := proto.Unmarshal(data, gen.Request); err != nil {
		glog.Fatal("unable to parse proto", err)
	}

	if len(gen.Request.FileToGenerate) == 0 {
		glog.Fatal("no files to generate")
	}

	var files []*plugin.CodeGeneratorResponse_File
	for _, file := range gen.Request.GetProtoFile() {
		if len(file.GetService()) > 0 {
			code := bytes.NewBuffer(nil)
			data := createTemplateData(file)
			if err := restTemplate.Execute(code, data); err != nil {
				glog.Fatal("unable to generate method", err)
			}

			if formatted, err := format.Source(code.Bytes()); err != nil {
				glog.Fatal(err)
			} else {
				name := file.GetName()
				ext := filepath.Ext(name)
				base := strings.TrimSuffix(name, ext)
				output := fmt.Sprintf("%s.rest.pb.go", base)
				files = append(files, &plugin.CodeGeneratorResponse_File{
					Name:    proto.String(output),
					Content: proto.String(string(formatted)),
				})
			}
		}
	}

	// Send back the results.
	data, err = proto.Marshal(&plugin.CodeGeneratorResponse{File: files})
	if err != nil {
		glog.Fatal("failed to marshal output proto", err)
	}
	_, err = os.Stdout.Write(data)
	if err != nil {
		glog.Fatal("failed to write output proto", err)
	}
}

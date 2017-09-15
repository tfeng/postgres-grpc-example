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
	"encoding/json"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net/http"
	"strings"
)

var (
	_ json.Token
	_ glog.Level
	_ mux.Route
	_ grpc.Server
)

func extractHeaders(ctx context.Context, req *http.Request) context.Context {
	var pairs []string
	for key, vals := range req.Header {
		for _, val := range vals {
			if strings.ToLower(key) == "authorization" {
				pairs = append(pairs, "authorization", val)
			}
		}
	}
	if len(pairs) == 0 {
		return ctx
	}
	md := metadata.Pairs(pairs...)
	return metadata.NewIncomingContext(ctx, md)
}

{{range $svc := .Services}}
func Create{{$svc.Service.GetName}}Router(ctx context.Context, impl {{$svc.Service.GetName}}Server, interceptor grpc.UnaryServerInterceptor, s *grpc.Server) (*mux.Router, error) {
	r := mux.NewRouter()
	{{range $m := $svc.Methods}}
	{{range $o := $m.HttpOpts}}
	r.HandleFunc({{$o.PathTemplate | printf "%q"}}, func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		if cn, ok := w.(http.CloseNotifier); ok {
			go func(done <-chan struct{}, closed <-chan bool) {
				select {
				case <-done:
				case <-closed:
					cancel()
				}
			}(ctx.Done(), cn.CloseNotify())
		}

		ctx = extractHeaders(ctx, r)

		var req {{$m.InputType}}
		json.NewDecoder(r.Body).Decode(&req)
		var resp *{{$m.OutputType}}
		var err error
		if interceptor == nil {
			resp, err = impl.{{$m.Method.GetName}}(ctx, &req)
		} else {
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				mi := req.(*{{$m.InputType}})
				r, err := impl.{{$m.Method.GetName}}(ctx, mi)
				return *r, err
			}
			if r, err := interceptor(ctx, &req, &grpc.UnaryServerInfo{Server: s, FullMethod: {{$m.Method.GetName | printf "%q"}}}, handler); err != nil {
				glog.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			} else {
				mo := r.({{$m.OutputType}})
				resp = &mo
			}
		}
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if buf, err := json.Marshal(resp); err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(buf); err != nil {
				glog.Error(err)
			}
		}
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

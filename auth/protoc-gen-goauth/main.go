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
	"github.com/tfeng/postgres-grpc-example/auth"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var authTemplate = template.Must(template.New("auth").Parse(`
package {{.Pkg}}

import (
	"fmt"
	jwt "github.com/dgrijalva/jwt-go"
	"golang.org/x/net/context"
)

var (
	_ fmt.Formatter
	_ jwt.Token
	_ context.Context
)

{{range $svc := .Services}}
{{range $md := $svc.Methods}}
{{if $md.AuthChecker.GetAuthenticated}}
func (r *{{.Request}}) isAuthenticated(ctx context.Context) bool {
	_, ok := ctx.Value("claims").(jwt.Claims)
	return ok
}
{{end}}

func (r *{{.Request}}) Authorize(ctx context.Context) error {
	{{if $md.AuthChecker.GetAuthenticated}}
	if !r.isAuthenticated(ctx) {
		return fmt.Errorf("not authenticated")
	}
	{{end}}
	return nil
}
{{end}}
{{end}}
`))

type Method struct {
	Method      *descriptor.MethodDescriptorProto
	Request     string
	AuthChecker *auth.AuthChecker
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
	var svcs []Service
	for _, svc := range file.GetService() {
		var mds []Method
		for _, md := range svc.GetMethod() {
			if ext, err := proto.GetExtension(md.GetOptions(), auth.E_Checker); err == nil {
				ac := ext.(*auth.AuthChecker)
				reqTypeParts := strings.Split(md.GetInputType(), ".")
				mds = append(mds, Method{
					md,
					reqTypeParts[len(reqTypeParts)-1],
					ac,
				})
			}
		}
		svcs = append(svcs, Service{
			svc,
			mds,
		})
	}
	return TemplateData{
		*file.Package,
		svcs,
	}
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
			if err := authTemplate.Execute(code, data); err != nil {
				glog.Fatal("unable to generate method")
			}

			if formatted, err := format.Source(code.Bytes()); err != nil {
				glog.Fatal(err)
			} else {
				name := file.GetName()
				ext := filepath.Ext(name)
				base := strings.TrimSuffix(name, ext)
				output := fmt.Sprintf("%s.auth.pb.go", base)
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

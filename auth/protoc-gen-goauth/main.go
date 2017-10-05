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
	{{if not .IsSamePackage}}
	"github.com/tfeng/postgres-grpc-example/auth"
	{{end}}
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	{{if not .IsSamePackage}}
	_ auth.AuthToken
	{{end}}
	_ context.Context
	_ codes.Code
	_ status.Status
)

{{range $svc := .Services}}
{{range $md := $svc.Methods}}
{{if (or ($md.AuthChecker.GetAuthenticated) ($md.AuthChecker.GetScope))}}
func (r *{{.Request}}) isAuthenticated(ctx context.Context) bool {
	_, ok := auth.GetAuthToken(ctx)
	return ok
}
{{end}}
{{if $md.AuthChecker.GetScope}}
func (r *{{.Request}}) HasScope(ctx context.Context) bool {
	token, _ := auth.GetAuthToken(ctx)
	return {{range $s := $md.AuthChecker.GetScope}}auth.HasScope(auth.Scope_{{$s}}, token) && {{end}}true
}
{{end}}

func (r *{{.Request}}) Authorize(ctx context.Context) error {
	{{if (or ($md.AuthChecker.GetAuthenticated) ($md.AuthChecker.GetScope))}}
	if !r.isAuthenticated(ctx) {
		return status.Error(codes.Unauthenticated, "Not authenticated")
	}
	{{end}}
	{{if $md.AuthChecker.GetScope}}
	if !r.HasScope(ctx) {
		return status.Error(codes.Unauthenticated, "Insufficient scope")
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
	Pkg           string
	Services      []Service
	IsSamePackage bool
}

type Params struct {
	IsAuthPackage bool
}

func parseParams(param string) Params {
	var p = Params{}
	parts := strings.Split(param, ",")
	for _, part := range parts {
		switch part {
		case "auth_package":
			p.IsAuthPackage = true
		}
	}
	return p
}

func createTemplateData(params Params, file *descriptor.FileDescriptorProto) TemplateData {
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
		params.IsAuthPackage,
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
	params := parseParams(gen.Request.GetParameter())
	for _, file := range gen.Request.GetProtoFile() {
		if len(file.GetService()) > 0 {
			code := bytes.NewBuffer(nil)
			data := createTemplateData(params, file)
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

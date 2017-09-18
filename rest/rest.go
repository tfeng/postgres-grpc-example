package rest

import (
	"github.com/golang/glog"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
	"strings"
)

const authorization = "authorization"

func extractHeaders(ctx context.Context, req *http.Request) context.Context {
	var pairs []string
	for key, vals := range req.Header {
		for _, val := range vals {
			if strings.ToLower(key) == authorization {
				pairs = append(pairs, authorization, val)
			}
		}
	}
	if len(pairs) == 0 {
		return ctx
	}
	md := metadata.Pairs(pairs...)
	return metadata.NewIncomingContext(ctx, md)
}

type implFunc func(context.Context, interface{}) (interface{}, error)

func HandleRequest(
	ctx context.Context,
	interceptor grpc.UnaryServerInterceptor,
	s *grpc.Server,
	w http.ResponseWriter,
	r *http.Request,
	req interface{},
	impl implFunc) {

	var resp interface{}
	var err error

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

	marshaler := runtime.JSONBuiltin{}
	if err := marshaler.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		glog.Error(err)
		runtime.HTTPError(ctx, &marshaler, w, r, status.Error(codes.InvalidArgument, "Invalid json body"))
		return
	}

	if interceptor == nil {
		resp, err = impl(ctx, req)
	} else {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			r, err := impl(ctx, req)
			return r, err
		}
		resp, err = interceptor(ctx, req, &grpc.UnaryServerInfo{Server: s, FullMethod: "CreateToken"}, handler)
	}
	if err != nil {
		runtime.HTTPError(ctx, &marshaler, w, r, err)
		return
	} else if buf, err := marshaler.Marshal(resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(buf)
	}
}

func HandleWrongContentType(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var marshaler = runtime.JSONBuiltin{}
	err := status.Error(codes.InvalidArgument, "Content-Type must be application/json")
	runtime.HTTPError(ctx, &marshaler, w, r, err)
}

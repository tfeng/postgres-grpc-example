package main

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	. "github.com/tfeng/postgres-grpc-example/datamodel"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"net/http"
)

func runGateway() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := RegisterUserServiceHandlerFromEndpoint(ctx, mux, "localhost:9090", opts)
	if err != nil {
		return err
	}

	return http.ListenAndServe(":8080", mux)
}

func main() {
	if err := runGateway(); err != nil {
		log.Fatal(err)
	}
}

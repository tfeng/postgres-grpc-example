package main

import (
	"flag"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/tfeng/postgres-grpc-example/auth"
	"github.com/tfeng/postgres-grpc-example/config"
	"github.com/tfeng/postgres-grpc-example/injection"
	"github.com/tfeng/postgres-grpc-example/models/user"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	math_rand "math/rand"
	"net"
	"net/http"
	"time"
)

func createTable() error {
	return db.CreateTable(&user.User{}, nil)
}

func dropTable() error {
	return db.DropTable(&user.User{}, nil)
}

func initialize() {
	flag.Parse()

	math_rand.Seed(time.Now().UTC().UnixNano())

	if err := dropTable(); err != nil {
		logger.Info("Unable to drop table", zap.Error(err))
	}

	if err := createTable(); err != nil {
		logger.Fatal("Unable to create table. ", zap.Error(err))
		return
	}
}

func createGrpcService() *grpc.Server {
	s := grpc.NewServer(grpc.StreamInterceptor(streamInterceptor), grpc.UnaryInterceptor(unaryInterceptor))
	user.RegisterUserServiceServer(s, &user.UserService{})
	auth.RegisterAuthServiceServer(s, &auth.AuthService{injection.GrantTypeHandlers})
	reflection.Register(s)
	return s
}

var (
	db                = config.Db
	logger            = config.Logger
	streamInterceptor = grpc_middleware.ChainStreamServer(
		grpc_ctxtags.StreamServerInterceptor(),
		grpc_validator.StreamServerInterceptor(),
		grpc_zap.StreamServerInterceptor(logger),
		auth.StreamServerInterceptor())
	unaryInterceptor = grpc_middleware.ChainUnaryServer(
		grpc_ctxtags.UnaryServerInterceptor(),
		grpc_validator.UnaryServerInterceptor(),
		grpc_zap.UnaryServerInterceptor(logger),
		auth.UnaryServerInterceptor())
)

func main() {
	initialize()

	s := createGrpcService()
	listener, err := net.Listen("tcp", ":9090")
	if err != nil {
		logger.Fatal("Unable to start service", zap.Error(err))
		return
	}
	go s.Serve(listener)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	r := mux.NewRouter()
	if ar, err := auth.CreateAuthServiceRouter(ctx, &auth.AuthService{injection.GrantTypeHandlers}, unaryInterceptor, s); err != nil {
		logger.Fatal("Unable to create auth router", zap.Error(err))
	} else if ur, err := user.CreateUserServiceRouter(ctx, &user.UserService{}, unaryInterceptor, s); err != nil {
		logger.Fatal("Unable to create user router", zap.Error(err))
	} else {
		r.Handle("/oauth/{_dummy:.*}", ar)
		r.Handle("/v1/users/{_dummy:.*}", ur)
		http.ListenAndServe(":8080", r)
	}
}

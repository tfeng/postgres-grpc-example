package main

import (
	"context"
	"encoding/json"
	"github.com/tfeng/postgres-grpc-example/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
)

const (
	address  = "localhost:9090"
	name     = "Thomas"
	password = "password"
	role     = "admin"
)

var (
	client = connect()
)

func connect() models.UserServiceClient {
	if conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure()); err != nil {
		panic(err)
	} else {
		return models.NewUserServiceClient(conn)
	}
}

func create() int64 {
	request := models.CreateRequest{Name: name, Password: password, Role: role}
	log.Println("create request", encode(request))
	if response, err := client.Create(context.Background(), &request); err != nil {
		panic(err)
	} else {
		log.Println("create response", encode(response))
		return response.Id
	}
}

func encode(obj interface{}) string {
	if b, err := json.Marshal(obj); err != nil {
		panic(err)
	} else {
		return string(b)
	}
}

func get(token string) *models.User {
	md := metadata.Pairs("authorization", "bearer "+token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	request := models.GetRequest{}
	log.Println("get request", encode(request))
	if response, err := client.Get(ctx, &request); err != nil {
		panic(err)
	} else {
		log.Println("get response", encode(response))
		return response
	}
}

func login(id int64) string {
	request := models.LoginRequest{Id: id, Password: password}
	log.Println("login request", encode(request))
	if response, err := client.Login(context.Background(), &request); err != nil {
		panic(err)
	} else {
		log.Println("login response", encode(response))
		return response.Token
	}
}

func main() {
	id := create()
	token := login(id)
	get(token)
}

package main

import (
	"context"
	"encoding/json"
	"github.com/tfeng/postgres-grpc-example/auth"
	"github.com/tfeng/postgres-grpc-example/models/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
)

const (
	address      = "localhost:9090"
	clientId     = "client"
	clientSecret = "password"
	username     = "amy"
	password     = "password"
)

var (
	authClient, userClient = connect()
)

func connect() (auth.AuthServiceClient, user.UserServiceClient) {
	if conn, err := grpc.Dial(address, grpc.WithInsecure()); err != nil {
		panic(err)
	} else {
		return auth.NewAuthServiceClient(conn), user.NewUserServiceClient(conn)
	}
}

func create(clientToken string) *user.User {
	md := metadata.Pairs("authorization", "bearer "+clientToken)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	request := user.CreateRequest{Username: username, Password: password}
	log.Println("create request", encode(request))
	if user, err := userClient.Create(ctx, &request); err != nil {
		panic(err)
	} else {
		log.Println("create response", encode(user))
		return user
	}
}

func encode(obj interface{}) string {
	if b, err := json.Marshal(obj); err != nil {
		panic(err)
	} else {
		return string(b)
	}
}

func get(userToken string) *user.User {
	md := metadata.Pairs("authorization", "bearer "+userToken)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	request := user.GetRequest{}
	log.Println("get request", encode(request))
	if user, err := userClient.Get(ctx, &request); err != nil {
		panic(err)
	} else {
		log.Println("get response", encode(user))
		return user
	}
}

func authorizeClient() string {
	request := auth.CreateTokenRequest{GrantType: "client_credentials", ClientId: clientId, ClientSecret: clientSecret}
	log.Println("authorize client request", encode(request))
	if response, err := authClient.CreateToken(context.Background(), &request); err != nil {
		panic(err)
	} else {
		log.Println("authorize client response", encode(response))
		return response.AccessToken
	}
}

func authorizeUser(clientToken string) string {
	md := metadata.Pairs("authorization", "bearer "+clientToken)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	request := auth.CreateTokenRequest{GrantType: "password", Username: username, Password: password}
	log.Println("authorize user request", encode(request))
	if response, err := authClient.CreateToken(ctx, &request); err != nil {
		panic(err)
	} else {
		log.Println("authorize user response", encode(response))
		return response.AccessToken
	}
}

func main() {
	clientToken := authorizeClient()
	create(clientToken)
	userToken := authorizeUser(clientToken)
	get(userToken)
}

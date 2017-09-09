package main

import (
	"github.com/go-pg/pg"
	. "github.com/tfeng/postgres-grpc-example/datamodel"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
)

func connect() *pg.DB {
	return pg.Connect(&pg.Options{
		Addr:     "localhost:5432",
		User:     "postgres",
		Password: "password",
	})
}

func createTable() error {
	return db.CreateTable(&User{}, nil)
}

func dropTable() error {
	return db.DropTable(&User{}, nil)
}

func initialize() {
	var err = dropTable()
	if err != nil {
		log.Println("Unable to drop table. ", err)
	}

	err = createTable()
	if err != nil {
		log.Fatal("Unable to create table. ", err)
	}
}

func insertUser(user *User) error {
	err := db.Insert(user)
	if err != nil {
		log.Println(err)
	}
	return err
}

func selectUser(user *User) error {
	err := db.Select(user)
	if err != nil {
		log.Println(err)
	}
	return err
}

func startService() error {
	listener, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	RegisterUserServiceServer(s, &UserService{})

	reflection.Register(s)
	return s.Serve(listener)
}

type UserService struct{}

func (userService *UserService) AddUser(context context.Context, request *AddUserRequest) (*AddUserResponse, error) {
	user := User{Name: request.Name}
	err := insertUser(&user)
	if err == nil {
		return &AddUserResponse{user.Id}, nil
	} else {
		return nil, err
	}
}

func (userService *UserService) GetUser(context context.Context, request *GetUserRequest) (*User, error) {
	user := User{Id: request.Id}
	err := selectUser(&user)
	if err == nil {
		return &user, nil
	} else {
		return nil, err
	}
}

var db = connect()

func main() {
	initialize()
	startService()

	/*var users []User
	err = db.Model(&users).Select()
	if err != nil {
		log.Fatal("Unable to select users; ", err)
	}
	log.Println(users)*/
}

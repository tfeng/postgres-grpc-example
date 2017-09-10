package main

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-pg/pg"
	"github.com/tfeng/postgres-grpc-example/models"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"strings"
	"time"
)

func connect() *pg.DB {
	return pg.Connect(&pg.Options{
		Addr:     "localhost:5432",
		User:     "postgres",
		Password: "password",
	})
}

func createTable() error {
	return db.CreateTable(&models.User{}, nil)
}

func dropTable() error {
	return db.DropTable(&models.User{}, nil)
}

func generateKey() *rsa.PrivateKey {
	if key, err := rsa.GenerateKey(rand.Reader, 1024); err != nil {
		log.Fatal(err)
		return nil
	} else {
		return key
	}
}

func initialize() {
	if err := dropTable(); err != nil {
		log.Println("Unable to drop table. ", err)
	}

	if err := createTable(); err != nil {
		log.Fatal("Unable to create table. ", err)
	}
}

func insertUser(user *models.User) error {
	err := db.Insert(user)
	if err != nil {
		log.Println(err)
	}
	return err
}

func selectUser(user *models.User) error {
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
	models.RegisterUserServiceServer(s, &UserService{})

	reflection.Register(s)
	return s.Serve(listener)
}

type UserService struct{}

func generateToken(user models.User) (string, error) {
	now := jwt.TimeFunc()
	claims := jwt.MapClaims{
		"id":   user.Id,
		"name": user.Name,
		"role": user.Role,
		"iat":  now.Unix(),
		"exp":  now.Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func parseToken(context context.Context) (jwt.Claims, error) {
	if md, ok := metadata.FromIncomingContext(context); !ok {
		return nil, errors.New("unable to parse metadata")
	} else {
		auths := md["authorization"]
		if len(auths) > 0 {
			auth := auths[0]
			if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				tokenString := strings.TrimSpace(auth[7:])
				if token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
						return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
					}
					return secretKey.Public(), nil
				}); err != nil {
					return nil, err
				} else {
					if err := token.Claims.Valid(); err != nil {
						return nil, err
					} else {
						return token.Claims, nil
					}
				}
			}
		}
		return nil, errors.New("no authorization header")
	}
}

func (userService *UserService) Create(context context.Context, request *models.CreateRequest) (*models.CreateResponse, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := models.User{Name: request.Name, HashedPassword: string(hashedPassword), Role: request.Role}
	if err := insertUser(&user); err != nil {
		return nil, err
	} else {
		return &models.CreateResponse{user.Id}, nil
	}
}

func (userService *UserService) Get(context context.Context, request *models.GetRequest) (*models.User, error) {
	if claims, err := parseToken(context); err != nil {
		return nil, err
	} else {
		mapClaims := claims.(jwt.MapClaims)
		user := models.User{Id: int64(mapClaims["id"].(float64))}
		if err := db.Select(&user); err != nil {
			return nil, err
		} else {
			user.HashedPassword = ""
			return &user, nil
		}
	}
}

func (userService *UserService) Login(context context.Context, request *models.LoginRequest) (*models.LoginResponse, error) {
	user := models.User{Id: request.Id}
	if err := selectUser(&user); err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(request.Password)); err != nil {
		return nil, err
	}

	if tokenString, err := generateToken(user); err != nil {
		return nil, err
	} else {
		return &models.LoginResponse{Token: tokenString}, nil
	}
}

var (
	db        = connect()
	secretKey = generateKey()
)

func main() {
	initialize()
	startService()
}

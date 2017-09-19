package user

import (
	"context"
	"github.com/dgrijalva/jwt-go"
	"github.com/tfeng/postgres-grpc-example/config"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

var (
	db         = config.Db
	privateKey = config.PrivateKey
)

type UserService struct{}

func (userService *UserService) Create(ctx context.Context, request *CreateRequest) (*CreateResponse, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid password")
	}

	u := User{Name: request.Name, HashedPassword: string(hashedPassword), Role: request.Role}
	if err := db.Insert(&u); err != nil {
		return nil, status.Error(codes.Internal, "Unable to create user")
	} else {
		return &CreateResponse{u.Id}, nil
	}
}

func (userService *UserService) Get(ctx context.Context, request *GetRequest) (*User, error) {
	claims, _ := ctx.Value("claims").(jwt.Claims)
	mapClaims := claims.(jwt.MapClaims)
	u := User{Id: int64(mapClaims["id"].(float64))}
	if err := db.Select(&u); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Unable to fetch user")
	} else {
		u.HashedPassword = ""
		return &u, nil
	}
}

func (userService *UserService) Login(ctx context.Context, request *LoginRequest) (*LoginResponse, error) {
	u := User{Id: request.Id}
	if err := db.Select(&u); err != nil {
		return nil, status.Error(codes.Unauthenticated, "Wrong user id or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.HashedPassword), []byte(request.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "Wrong user id or password")
	}

	if tokenString, err := generateToken(u); err != nil {
		return nil, status.Error(codes.Unauthenticated, "Unable to authenticate")
	} else {
		return &LoginResponse{Token: tokenString}, nil
	}
}

func generateToken(u User) (string, error) {
	now := jwt.TimeFunc()
	claims := jwt.MapClaims{
		"id":   u.Id,
		"name": u.Name,
		"role": u.Role,
		"iat":  now.Unix(),
		"exp":  now.Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

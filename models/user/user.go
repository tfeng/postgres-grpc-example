package user

import (
	"context"
	"github.com/tfeng/postgres-grpc-example/auth"
	"github.com/tfeng/postgres-grpc-example/config"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	db = config.Db
)

type UserStore struct{}

func (h *UserStore) GetUserInfo(username string) (*auth.UserInfo, error) {
	u := User{Id: username}
	if err := db.Select(&u); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Unable to fetch user")
	} else {
		return &auth.UserInfo{u.HashedPassword, []auth.Scope{auth.Scope_user_profile}}, nil
	}
}

type UserService struct{}

func (userService *UserService) Create(ctx context.Context, request *CreateRequest) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid password")
	}

	u := User{Id: request.Username, HashedPassword: string(hashedPassword)}
	if err := db.Insert(&u); err != nil {
		return nil, status.Error(codes.Internal, "Unable to create user")
	} else {
		u.HashedPassword = ""
		return &u, nil
	}
}

func (userService *UserService) Get(ctx context.Context, request *GetRequest) (*User, error) {
	token, _ := auth.GetAuthToken(ctx)
	u := User{Id: token.UserId}
	if err := db.Select(&u); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Unable to fetch user")
	} else {
		u.HashedPassword = ""
		return &u, nil
	}
}

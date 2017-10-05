package auth

import (
	"context"
	"github.com/golang/protobuf/ptypes"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math"
	"strings"
	"time"
)

type UserInfo struct {
	HashedPassword string
	Scope          []Scope
}

type userStore interface {
	GetUserInfo(string) (*UserInfo, error)
}

const USER_TOKEN_EXPIRATION = time.Hour * 24

type UserPasswordGrantTypeHandler struct {
	UserStore userStore
}

func (h *UserPasswordGrantTypeHandler) createAuthToken(ctx context.Context, r *CreateTokenRequest) (*AuthToken, error) {
	var err error
	var authToken AuthToken
	now := time.Now()

	if r.GrantType != GrantType_password.String() {
		return nil, status.Error(codes.Unauthenticated, "Unexpected grant type")
	}

	clientAuthToken, ok := GetAuthToken(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "Not authenticated")
	}
	if !HasScope(Scope_user_authorize, clientAuthToken) {
		return nil, status.Error(codes.Unauthenticated, "Insufficient scope")
	}
	authToken.ClientId = clientAuthToken.ClientId

	var password string
	authToken.UserId, password, err = h.getUsernamePassword(ctx, r)
	userInfo, err := h.UserStore.GetUserInfo(authToken.UserId)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Incorrect user id or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(userInfo.HashedPassword), []byte(password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "Incorrect user id or password")
	}

	authToken.Scope = userInfo.Scope

	authToken.Access, err = generateToken()
	if err != nil {
		return nil, err
	}
	authToken.AccessCreationTime, err = ptypes.TimestampProto(now)
	if err != nil {
		return nil, err
	}
	authToken.AccessExpirationTime, err = ptypes.TimestampProto(now.Add(USER_TOKEN_EXPIRATION))
	if err != nil {
		return nil, err
	}

	authToken.Refresh, err = generateToken()
	if err != nil {
		return nil, err
	}

	addAuthToken(authToken)

	return &authToken, nil
}

func (h *UserPasswordGrantTypeHandler) getUsernamePassword(ctx context.Context, r *CreateTokenRequest) (string, string, error) {
	var username, password string
	if s, err := grpc_auth.AuthFromMD(ctx, "Basic"); err == nil {
		if u, p, ok := parseBasicAuth(s); ok {
			username = u
			password = p
		} else {
			return "", "", status.Error(codes.Unauthenticated, "Invalid basic auth")
		}
	} else {
		username = r.Username
		password = r.Password
	}
	return username, password, nil
}

func (h *UserPasswordGrantTypeHandler) CreateToken(ctx context.Context, r *CreateTokenRequest) (*CreateTokenResponse, error) {
	authToken, err := h.createAuthToken(ctx, r)
	if err != nil {
		return nil, err
	}

	resp := CreateTokenResponse{TokenType: "bearer", AccessToken: authToken.Access}

	var scopeNames []string
	for _, scope := range authToken.Scope {
		scopeNames = append(scopeNames, Scope_name[int32(scope)])
	}
	resp.Scope = strings.Join(scopeNames, " ")

	t, err := ptypes.Timestamp(authToken.AccessExpirationTime)
	if err != nil {
		return nil, err
	}
	resp.ExpiresIn = int32(math.Ceil(t.Sub(time.Now()).Seconds()))

	return &resp, nil
}

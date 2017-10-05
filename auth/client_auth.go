package auth

import (
	"context"
	"github.com/golang/protobuf/ptypes"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math"
	"strings"
	"time"
)

type ClientInfo struct {
	Secret string
	Scope  []Scope
}

var clients = map[string]ClientInfo{
	"client": {"password", []Scope{Scope_user_creation, Scope_user_authorize}},
}

type clientStore interface {
	GetClientInfo(string) (*ClientInfo, error)
}

const CLIENT_TOKEN_EXPIRATION = time.Hour * 24

type ClientCredentialsGrantTypeHandler struct {
	ClientStore clientStore
}

func (h *ClientCredentialsGrantTypeHandler) createAuthToken(ctx context.Context, r *CreateTokenRequest) (*AuthToken, error) {
	var err error
	var authToken AuthToken
	now := time.Now()

	if r.GrantType != GrantType_client_credentials.String() {
		return nil, status.Error(codes.Unauthenticated, "Unexpected grant type")
	}

	var password string
	authToken.ClientId, password, err = h.getUsernamePassword(ctx, r)
	clientInfo, ok := clients[authToken.ClientId]
	if !ok || password != clientInfo.Secret {
		return nil, status.Error(codes.Unauthenticated, "Incorrect client id or secret")
	}

	authToken.Scope = clientInfo.Scope

	authToken.Access, err = generateToken()
	if err != nil {
		return nil, err
	}
	authToken.AccessCreationTime, err = ptypes.TimestampProto(now)
	if err != nil {
		return nil, err
	}
	authToken.AccessExpirationTime, err = ptypes.TimestampProto(now.Add(CLIENT_TOKEN_EXPIRATION))
	if err != nil {
		return nil, err
	}

	addAuthToken(authToken)

	return &authToken, nil
}

func (h *ClientCredentialsGrantTypeHandler) getUsernamePassword(ctx context.Context, r *CreateTokenRequest) (string, string, error) {
	var username, password string
	if s, err := grpc_auth.AuthFromMD(ctx, "Basic"); err == nil {
		if u, p, ok := parseBasicAuth(s); ok {
			username = u
			password = p
		} else {
			return "", "", status.Error(codes.Unauthenticated, "Invalid basic auth")
		}
	} else {
		username = r.ClientId
		password = r.ClientSecret
	}
	return username, password, nil
}

func (h *ClientCredentialsGrantTypeHandler) CreateToken(ctx context.Context, r *CreateTokenRequest) (*CreateTokenResponse, error) {
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

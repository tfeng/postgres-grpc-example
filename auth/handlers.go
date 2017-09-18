package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

const (
	clientId     = "client"
	clientSecret = "password"
	expiration   = 86400
)

type grantTypeHandler interface {
	CreateToken(context.Context, *CreateTokenRequest) (*CreateTokenResponse, error)
}

type clientCredentialsGrantTypeHandler struct{}

func (h *clientCredentialsGrantTypeHandler) parseBasicAuth(s string) (username, password string, ok bool) {
	c, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return
	}
	cs := string(c)
	idx := strings.IndexByte(cs, ':')
	if idx < 0 {
		return
	}
	return cs[:idx], cs[idx+1:], true
}

func (h *clientCredentialsGrantTypeHandler) createResponse(scopes []Scope) (*CreateTokenResponse, error) {
	var r CreateTokenResponse
	if token, err := GenerateToken(); err != nil {
		return nil, err
	} else {
		var scopeNames []string
		for _, scope := range scopes {
			scopeNames = append(scopeNames, Scope_name[int32(scope)])
		}
		r.Scope = strings.Join(scopeNames, " ")
		r.AccessToken = token
		r.ExpiresIn = expiration
		r.TokenType = CreateTokenResponse_Bearer
		return &r, nil
	}
}

func (h *clientCredentialsGrantTypeHandler) parseScopes(s string) ([]Scope, error) {
	var scopes []Scope
	parts := strings.Split(s, " ")
	for _, part := range parts {
		if val, ok := Scope_value[part]; ok {
			scopes = append(scopes, Scope(val))
		} else {
			return nil, fmt.Errorf("Scope %s does not exist", part)
		}
	}
	if len(scopes) == 0 {
		return nil, fmt.Errorf("No scope specified")
	}
	return scopes, nil
}

func (h *clientCredentialsGrantTypeHandler) CreateToken(ctx context.Context, r *CreateTokenRequest) (*CreateTokenResponse, error) {
	var username, password string
	if s, err := grpc_auth.AuthFromMD(ctx, "Basic"); err == nil {
		if u, p, ok := h.parseBasicAuth(s); ok {
			username = u
			password = p
		} else {
			return nil, status.Error(codes.Unauthenticated, "Invalid basic auth")
		}
	} else {
		username = r.ClientId
		password = r.ClientSecret
	}
	if username == clientId && password == clientSecret {
		if scopes, err := h.parseScopes(r.Scope); err != nil {
			return nil, status.Error(codes.InvalidArgument, "Invalid scope")
		} else {
			return h.createResponse(scopes)
		}
	} else {
		return nil, status.Error(codes.Unauthenticated, "Incorrect client id or secret")
	}
}

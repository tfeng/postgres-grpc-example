package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

type ClientInfo struct {
	secret     string
	expiration time.Duration
	scopes     []Scope
}

type UserInfo struct {
	password string
}

var clients = map[string]ClientInfo{
	"client": {"password", time.Hour * 24, []Scope{Scope_public, Scope_profile}},
}

var users = map[string]UserInfo{
	"tfeng": {"password"},
}

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

func (h *clientCredentialsGrantTypeHandler) createResponse(client string) (*CreateTokenResponse, error) {
	token, err := GenerateToken()
	if err != nil {
		return nil, err
	}

	clientInfo, ok := clients[client]
	if !ok {
		return nil, errors.New("Client " + client + " does not exist")
	}

	var r CreateTokenResponse
	var scopeNames []string
	for _, scope := range clientInfo.scopes {
		scopeNames = append(scopeNames, Scope_name[int32(scope)])
	}
	r.Scope = strings.Join(scopeNames, " ")
	r.AccessToken = token
	r.ExpiresIn = int32(clientInfo.expiration.Seconds())
	r.TokenType = "bearer"
	return &r, nil
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

	clientInfo, ok := clients[username]
	if !ok || password != clientInfo.secret {
		return nil, status.Error(codes.Unauthenticated, "Incorrect client id or secret")
	}
	return h.createResponse(username)
}

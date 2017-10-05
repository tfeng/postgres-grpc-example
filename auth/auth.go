package auth

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

var (
	tokenStore        = make(map[string]AuthToken)
	refreshTokenStore = make(map[string]AuthToken)
)

type authorizable interface {
	Authorize(context.Context) error
}

type GrantTypeHandler interface {
	CreateToken(context.Context, *CreateTokenRequest) (*CreateTokenResponse, error)
}

type streamWrapper struct {
	grpc.ServerStream
}

func (s *streamWrapper) RecvMsg(m interface{}) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}
	if auth, ok := m.(authorizable); ok {
		if err := auth.Authorize(s.ServerStream.Context()); err != nil {
			return err
		}
	}
	return nil
}

func parseBasicAuth(s string) (username, password string, ok bool) {
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

func generateToken() (string, error) {
	b := make([]byte, 33)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func addAuthToken(authToken AuthToken) {
	tokenStore[authToken.Access] = authToken
	if authToken.Refresh != "" {
		refreshTokenStore[authToken.Refresh] = authToken
	}
}

func extractAuthToken(ctx context.Context) *AuthToken {
	var err error
	var token string
	token, err = grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil
	}
	authToken, ok := tokenStore[token]
	if !ok {
		return nil
	}
	return &authToken
}

func getAuthToken(access string) AuthToken {
	return tokenStore[access]
}

func HasScope(scope Scope, authToken *AuthToken) bool {
	if authToken != nil {
		for _, s := range authToken.Scope {
			if s == scope {
				return true
			}
		}
	}
	return false
}

func GetAuthToken(ctx context.Context) (*AuthToken, bool) {
	authToken, ok := ctx.Value("token").(*AuthToken)
	return authToken, ok
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapper := &streamWrapper{stream}
		return handler(srv, wrapper)
	}
}

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		token := extractAuthToken(ctx)
		if token != nil {
			ctx = context.WithValue(ctx, "token", token)
		}
		if auth, ok := req.(authorizable); ok {
			if err := auth.Authorize(ctx); err != nil {
				return nil, err
			}
		}
		return handler(ctx, req)
	}
}

type AuthService struct {
	GrantTypeHandlers map[string]GrantTypeHandler
}

func (c *AuthService) CreateToken(ctx context.Context, r *CreateTokenRequest) (*CreateTokenResponse, error) {
	if handler, ok := c.GrantTypeHandlers[r.GrantType]; ok {
		return handler.CreateToken(ctx, r)
	} else {
		return nil, status.Error(codes.InvalidArgument, "Unknown grant type")
	}
}

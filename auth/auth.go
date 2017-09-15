package auth

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type authorizable interface {
	Authorize(context.Context) error
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

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapper := &streamWrapper{stream}
		return handler(srv, wrapper)
	}
}

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if auth, ok := req.(authorizable); ok {
			if err := auth.Authorize(ctx); err != nil {
				return nil, err
			}
		}
		return handler(ctx, req)
	}
}

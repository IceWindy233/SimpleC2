package main

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// NewAuthInterceptor returns a gRPC unary server interceptor that validates an API key.
func NewAuthInterceptor(expectedAPIKey string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader, ok := md["authorization"]
		if !ok || len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		parts := strings.Split(authHeader[0], " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		if parts[1] != expectedAPIKey {
			return nil, status.Error(codes.Unauthenticated, "invalid API key")
		}

		// If the API key is valid, proceed with the original handler.
		return handler(ctx, req)
	}
}


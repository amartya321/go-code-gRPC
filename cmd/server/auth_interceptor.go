package main

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func authUnaryInterceptor(expectedToken string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		vals := md.Get("authorization")
		if len(vals) == 0 || strings.TrimSpace(vals[0]) == "" {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}
		if vals[0] != "Bearer "+expectedToken {
			return nil, status.Error(codes.PermissionDenied, "invalid authorization token")
		}
		return handler(ctx, req)
	}
}

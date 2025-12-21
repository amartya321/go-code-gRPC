package main

import (
	"context"
	"log"
	"net"
	"time"

	hellov1 "grpc-lab/gen/hello/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type greeterServer struct {
	hellov1.UnimplementedGreeterServer
}

func (s *greeterServer) SayHello(ctx context.Context, req *hellov1.HelloRequest) (*hellov1.HelloReply, error) {
	name := req.GetFullName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "full_name is required")
	}
	select {
	case <-time.After(3 * time.Second):
		// continue
	case <-ctx.Done():
		switch ctx.Err() {
		case context.DeadlineExceeded:
			return nil, status.Error(codes.DeadlineExceeded, "deadline exceeded")
		case context.Canceled:
			return nil, status.Error(codes.Canceled, "request cancelled by client")
		default:
			return nil, status.Error(codes.Canceled, "request ended")
		}
	}
	return &hellov1.HelloReply{Message: "Hello, " + name + " 👋"}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	hellov1.RegisterGreeterServer(grpcServer, &greeterServer{})

	log.Println("gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

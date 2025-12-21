package main

import (
	"context"
	"log"
	"net"

	hellov1 "grpc-lab/gen/hello/v1"

	"google.golang.org/grpc"
)

type greeterServer struct {
	hellov1.UnimplementedGreeterServer
}

func (s *greeterServer) SayHello(ctx context.Context, req *hellov1.HelloRequest) (*hellov1.HelloReply, error) {
	name := req.GetFullName()
	if name == "" {
		name = "stranger"
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

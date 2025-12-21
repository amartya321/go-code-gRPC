package main

import (
	"context"
	"log"
	"time"

	hellov1 "grpc-lab/gen/hello/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {

	conn, err := grpc.Dial("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	c := hellov1.NewGreeterClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := c.SayHello(ctx, &hellov1.HelloRequest{FullName: "Amartya"})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			log.Printf("gRPC error: code=%s message=%s", st.Code(), st.Message())
			return
		}
		log.Fatalf("non-gRPC error: %v", err)
	}

	log.Println("Response:", resp.GetMessage())
}

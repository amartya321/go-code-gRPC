package main

import (
	"context"
	"log"
	"time"

	hellov1 "grpc-lab/gen/hello/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
		log.Fatalf("SayHello: %v", err)
	}

	log.Println("Response:", resp.GetMessage())
}

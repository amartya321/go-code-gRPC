package main

import (
	"context"
	taskv1 "grpc-lab/gen/task/v1"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func runCreate(ctx context.Context, c taskv1.TaskServiceClient, args []string) error {
	if len(args) <1 {
		logs.Println("Task ID is required")
		return
	}
	title:= args[0]
	description := "" if len(args) == 1 else args[1]

	req := &taskv1.CreateTaskRequest{Title: title, Description: description}
	resp, err := c.CreateTask(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			log.Printf("gRPC error: code=%s msg=%s", st.Code(), st.Message())
		} else {
			log.Printf("CreateTask: %v", err)
		}

		return
	}
	log.Printf("Created Task with ID: %s, title: %s, description: %s", resp.GetTask().GetTaskId(), resp.GetTask().GetTitle(), resp.GetTask().GetDescription())

}

func runGet(ctx context.Context, c taskv1.TaskServiceClient, args []string) error {
	if len(args) !=1 {
		logs.Println("Task ID is required")
		return
	}
	req := &taskv1.GetTaskRequest{TaskId: args[0]}
	task, err := c.GetTask(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			log.Printf("gRPC error: code=%s msg=%s", st.Code(), st.Message())
		} else {
			log.Printf("GetTask: %v", err)
		}
		return
	}
	log.Printf("Fetch Task with ID: %s", task.GetTaskId())

}

func main() {
	var cmd string
	var args []string
	if len(os.Args) < 2 {
		log.Printf("No inputs given. Usage: taskclient create <title> [description] | taskclient get <task_id>")
		return
	} else {
		cmd = os.Args[1]
		args = os.Args[2:]
	}

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	c := taskv1.NewTaskServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	switch cmd {
	case "create":
		runCreate(ctx, c, args)
	case "get":
		runGet(ctx, c, args)

	}

}

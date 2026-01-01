package main

import (
	"context"
	"fmt"
	taskv1 "grpc-lab/gen/task/v1"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func runCreate(ctx context.Context, c taskv1.TaskServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("task title is required")
	}
	title := args[0]
	var description string
	if len(args) == 1 {
		description = ""
	} else {
		description = strings.Join(args[1:], " ")
	}

	req := &taskv1.CreateTaskRequest{Title: title, Description: description}
	resp, err := c.CreateTask(ctx, req)
	if err != nil {
		return err
	}
	log.Printf("Created Task with ID: %s, title: %s, description: %s", resp.GetTask().GetTaskId(), resp.GetTask().GetTitle(), resp.GetTask().GetDescription())
	return nil
}

func runGet(ctx context.Context, c taskv1.TaskServiceClient, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("task id is required")
	}
	req := &taskv1.GetTaskRequest{TaskId: args[0]}
	task, err := c.GetTask(ctx, req)
	if err != nil {
		return err
	}
	log.Printf("Fetched Task with ID: %s Title: %s Description: %s", task.GetTaskId(), task.GetTitle(), task.GetDescription())
	return nil
}

func main() {
	var cmd string
	var args []string
	var err error
	if len(os.Args) < 2 {
		log.Printf("No inputs given. Usage: taskclient create <title> [description] | taskclient get <task_id>")
		return
	} else {
		cmd = os.Args[1]
		args = os.Args[2:]
	}

	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial: %v", err)
		return
	}
	defer conn.Close()
	c := taskv1.NewTaskServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	switch cmd {
	case "create":
		err = runCreate(ctx, c, args)
	case "get":
		err = runGet(ctx, c, args)
	default:
		log.Printf("Unknown command: %s. Usage: taskclient create <title> [description] | taskclient get <task_id>", cmd)
		return
	}
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			log.Fatalf("command %s failed: code=%s msg=%s", cmd, st.Code(), st.Message())
		} else {
			log.Fatalf("command %s failed: %s", cmd, err)
		}
	}

}

package main

import (
	"context"
	"fmt"
	taskv1 "grpc-lab/gen/task/v1"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type tokenCreds string

func (t tokenCreds) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + string(t)}, nil
}

func (t tokenCreds) RequireTransportSecurity() bool {
	return false
}

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

func runList(ctx context.Context, c taskv1.TaskServiceClient, args []string) error {
	page_size := 10
	page_token := ""
	var err error
	if len(args) >= 1 {
		page_size, err = strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid page_size: %s", args[0])
		}
	}
	if len(args) == 2 {
		page_token = args[1]
	}
	req := &taskv1.ListTasksRequest{PageSize: int32(page_size), PageToken: page_token}
	resp, err := c.ListTasks(ctx, req)
	if err != nil {
		return err
	}
	for _, task := range resp.GetTasks() {
		log.Printf("Task ID: %s Title: %s Description: %s,", task.GetTaskId(), task.GetTitle(), task.GetDescription())
	}
	log.Printf("Next Page Token %s", resp.GetNextPageToken())
	return nil

}

func runWatch(ctx context.Context, c taskv1.TaskServiceClient, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("task id is required")
	}
	stream, err := c.WatchTask(ctx, &taskv1.WatchTaskRequest{TaskId: args[0]})
	if err != nil {
		return err
	}
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		log.Printf("Task Event: status=%v at=%s message=%s", event.GetStatus().String(), event.GetAt().AsTime().String(), event.GetMessage())
	}
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

	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithPerRPCCredentials(tokenCreds("devtoken")))
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
	case "list":
		err = runList(ctx, c, args)
	case "watch":
		err = runWatch(ctx, c, args)
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

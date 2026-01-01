package main

import (
	"context"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	hellov1 "grpc-lab/gen/hello/v1"
	taskv1 "grpc-lab/gen/task/v1"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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

type TaskServiceServer struct {
	taskv1.UnimplementedTaskServiceServer

	mu        sync.RWMutex
	taskMap   map[string]*taskv1.Task
	taskSlice []*taskv1.Task
}

func NewTaskServiceServer() *TaskServiceServer {
	return &TaskServiceServer{
		taskMap:   make(map[string]*taskv1.Task),
		taskSlice: make([]*taskv1.Task, 0),
	}
}

func (s *TaskServiceServer) CreateTask(ctx context.Context, req *taskv1.CreateTaskRequest) (res *taskv1.CreateTaskResponse, err error) {
	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	id := uuid.New()
	now := timestamppb.New(time.Now())
	task := &taskv1.Task{
		TaskId:      id.String(),
		Title:       title,
		Description: strings.TrimSpace(req.GetDescription()),
		CreatedAt:   now,
		UpdatedAt:   now,
		Status:      taskv1.TaskStatus_TASK_STATUS_PENDING,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.taskMap[task.TaskId] = task
	s.taskSlice = append(s.taskSlice, task)
	return &taskv1.CreateTaskResponse{Task: task}, nil
}

func (s *TaskServiceServer) GetTask(ctx context.Context, req *taskv1.GetTaskRequest) (res *taskv1.Task, err error) {
	id := strings.TrimSpace(req.GetTaskId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "task_id is required")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.taskMap[id]
	if !ok {
		return nil, status.Error(codes.NotFound, "task not found with id "+id)
	}
	return task, nil
}

func main() {

	s := NewTaskServiceServer()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	taskv1.RegisterTaskServiceServer(grpcServer, s)

	log.Println("gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

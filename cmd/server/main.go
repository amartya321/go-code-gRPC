package main

import (
	"context"
	"log"
	"net"
	"strconv"
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
	failNext  bool
}

func (s *TaskServiceServer) FailNextUnavailable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = true

}

func NewTaskServiceServer() *TaskServiceServer {
	return &TaskServiceServer{
		taskMap:   make(map[string]*taskv1.Task),
		taskSlice: make([]*taskv1.Task, 0),
	}
}

func (s *TaskServiceServer) CreateTask(ctx context.Context, req *taskv1.CreateTaskRequest) (res *taskv1.CreateTaskResponse, err error) {
	if s.failNext {
		s.mu.Lock()
		s.failNext = false
		s.mu.Unlock()
		return nil, status.Error(codes.Unavailable, "simulated failure")
	}
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

func (s *TaskServiceServer) CreateTaskWithId(ctx context.Context, req *taskv1.CreateTaskWithIdRequest) (res *taskv1.CreateTaskResponse, err error) {
	task_id := strings.TrimSpace(req.GetTaskId())
	if task_id == "" {
		return nil, status.Error(codes.InvalidArgument, "task_id is required")
	}
	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.taskMap[task_id]; exists {
		return nil, status.Error(codes.AlreadyExists, "task with id "+task_id+" already exists")
	}
	now := timestamppb.New(time.Now())
	task := &taskv1.Task{
		TaskId:      task_id,
		Title:       title,
		Description: strings.TrimSpace(req.GetDescription()),
		CreatedAt:   now,
		UpdatedAt:   now,
		Status:      taskv1.TaskStatus_TASK_STATUS_PENDING,
	}
	s.taskMap[task.TaskId] = task
	s.taskSlice = append(s.taskSlice, task)
	return &taskv1.CreateTaskResponse{Task: task}, nil
}

func (s *TaskServiceServer) ListTasks(ctx context.Context, req *taskv1.ListTasksRequest) (res *taskv1.ListTasksResponse, err error) {
	page_size := req.GetPageSize()
	if page_size <= 0 {
		page_size = 10
	} else if page_size > 100 {
		page_size = 100
	}
	offset := 0
	page_token := req.GetPageToken()
	if page_token == "" {
		offset = 0
	} else {
		offset, err = strconv.Atoi(page_token)
		if err != nil || offset < 0 {
			return nil, status.Error(codes.InvalidArgument, "invalid page_token")
		}
		if offset >= len(s.taskSlice) {
			return &taskv1.ListTasksResponse{
				Tasks:         []*taskv1.Task{},
				NextPageToken: "",
			}, nil
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	start := offset
	end := min(offset+int(page_size), len(s.taskSlice))
	res = &taskv1.ListTasksResponse{
		Tasks: make([]*taskv1.Task, 0),
	}
	res.Tasks = s.taskSlice[start:end]
	if end >= len(s.taskSlice) {
		res.NextPageToken = ""
	} else {
		res.NextPageToken = strconv.Itoa(end)
	}
	return res, nil

}

func main() {

	s := NewTaskServiceServer()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(authUnaryInterceptor("devtoken")))
	taskv1.RegisterTaskServiceServer(grpcServer, s)

	log.Println("gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

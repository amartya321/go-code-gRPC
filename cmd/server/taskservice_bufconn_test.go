package main

import (
	"context"
	"net"
	"strconv"
	"testing"

	taskv1 "grpc-lab/gen/task/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func ctxWithAuth(token string) context.Context {
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
}
func newBufconnClient(t *testing.T) (taskv1.TaskServiceClient, func()) {
	t.Helper()

	lis := bufconn.Listen(bufSize)

	svc := NewTaskServiceServer()

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(authUnaryInterceptor("devtoken")))
	taskv1.RegisterTaskServiceServer(grpcServer, svc)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("DialContext(bufnet) failed: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = lis.Close()
	}

	return taskv1.NewTaskServiceClient(conn), cleanup
}

func newBufconnClientWithServer(t *testing.T) (taskv1.TaskServiceClient, *TaskServiceServer, func()) {
	t.Helper()

	lis := bufconn.Listen(bufSize)

	svc := NewTaskServiceServer()

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(authUnaryInterceptor("devtoken")))
	taskv1.RegisterTaskServiceServer(grpcServer, svc)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.DialContext(
		ctxWithAuth("devtoken"),
		"bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("DialContext(bufnet) failed: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = lis.Close()
	}

	return taskv1.NewTaskServiceClient(conn), svc, cleanup
}

func TestTaskService_CreateThenGet(t *testing.T) {
	client, cleanup := newBufconnClient(t)
	defer cleanup()

	// 1) Create
	createResp, err := client.CreateTask(ctxWithAuth("devtoken"), &taskv1.CreateTaskRequest{
		Title:       "buy milk",
		Description: "tonight",
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	created := createResp.GetTask()
	if created.GetTaskId() == "" {
		t.Fatalf("expected task_id to be set")
	}
	if created.GetTitle() != "buy milk" {
		t.Fatalf("expected title %q, got %q", "buy milk", created.GetTitle())
	}

	// 2) Get
	got, err := client.GetTask(ctxWithAuth("devtoken"), &taskv1.GetTaskRequest{
		TaskId: created.GetTaskId(),
	})
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if got.GetTaskId() != created.GetTaskId() {
		t.Fatalf("expected same task_id back")
	}
	if got.GetDescription() != "tonight" {
		t.Fatalf("expected description %q, got %q", "tonight", got.GetDescription())
	}
}

func TestTaskService_Get_NotFound(t *testing.T) {
	client, cleanup := newBufconnClient(t)
	defer cleanup()

	_, err := client.GetTask(ctxWithAuth("devtoken"), &taskv1.GetTaskRequest{
		TaskId: "does-not-exist",
	})

	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error")
	}

	if st.Code() != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", st.Code())
	}
}

func TestTaskService_List_Pagination(t *testing.T) {
	client, cleanup := newBufconnClient(t)
	defer cleanup()

	// Create 5 tasks in order.
	var ids []string
	for i := 1; i <= 5; i++ {
		resp, err := client.CreateTask(ctxWithAuth("devtoken"), &taskv1.CreateTaskRequest{
			Title: "t" + strconv.Itoa(i),
		})
		if err != nil {
			t.Fatalf("CreateTask(%d) failed: %v", i, err)
		}
		ids = append(ids, resp.GetTask().GetTaskId())
	}

	// Page 1: size 2
	page1, err := client.ListTasks(ctxWithAuth("devtoken"), &taskv1.ListTasksRequest{
		PageSize:  2,
		PageToken: "",
	})
	if err != nil {
		t.Fatalf("ListTasks page1 failed: %v", err)
	}
	if len(page1.GetTasks()) != 2 {
		t.Fatalf("expected 2 tasks on page1, got %d", len(page1.GetTasks()))
	}
	if page1.GetTasks()[0].GetTaskId() != ids[0] || page1.GetTasks()[1].GetTaskId() != ids[1] {
		t.Fatalf("page1 order mismatch")
	}
	if page1.GetNextPageToken() == "" {
		t.Fatalf("expected next_page_token on page1")
	}

	// Page 2: size 2
	page2, err := client.ListTasks(ctxWithAuth("devtoken"), &taskv1.ListTasksRequest{
		PageSize:  2,
		PageToken: page1.GetNextPageToken(),
	})
	if err != nil {
		t.Fatalf("ListTasks page2 failed: %v", err)
	}
	if len(page2.GetTasks()) != 2 {
		t.Fatalf("expected 2 tasks on page2, got %d", len(page2.GetTasks()))
	}
	if page2.GetTasks()[0].GetTaskId() != ids[2] || page2.GetTasks()[1].GetTaskId() != ids[3] {
		t.Fatalf("page2 order mismatch")
	}
	if page2.GetNextPageToken() == "" {
		t.Fatalf("expected next_page_token on page2")
	}

	// Page 3: size 2 (should contain last 1)
	page3, err := client.ListTasks(ctxWithAuth("devtoken"), &taskv1.ListTasksRequest{
		PageSize:  2,
		PageToken: page2.GetNextPageToken(),
	})
	if err != nil {
		t.Fatalf("ListTasks page3 failed: %v", err)
	}
	if len(page3.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page3, got %d", len(page3.GetTasks()))
	}
	if page3.GetTasks()[0].GetTaskId() != ids[4] {
		t.Fatalf("page3 order mismatch")
	}
	if page3.GetNextPageToken() != "" {
		t.Fatalf("expected empty next_page_token on last page, got %q", page3.GetNextPageToken())
	}
}

func TestTaskService_Create_InvalidArgument(t *testing.T) {
	client, cleanup := newBufconnClient(t)
	defer cleanup()
	cases := []struct {
		name string
		req  *taskv1.CreateTaskRequest
		code codes.Code
	}{
		{
			name: "empty title",
			req:  &taskv1.CreateTaskRequest{Title: ""},
			code: codes.InvalidArgument,
		},
		{
			name: "whitespace title",
			req:  &taskv1.CreateTaskRequest{Title: "   "},
			code: codes.InvalidArgument,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.CreateTask(ctxWithAuth("devtoken"), tc.req)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if status.Code(err) != tc.code {
				t.Fatalf("expected code %v, got %v", tc.code, status.Code(err))
			}
		})
	}
}

func TestTaskService_Get_InvalidArgument(t *testing.T) {
	client, cleanup := newBufconnClient(t)
	defer cleanup()
	cases := []struct {
		name string
		req  *taskv1.GetTaskRequest
		code codes.Code
	}{
		{
			name: "empty task_id",
			req:  &taskv1.GetTaskRequest{TaskId: ""},
			code: codes.InvalidArgument,
		},
		{
			name: "whitespace task_id",
			req:  &taskv1.GetTaskRequest{TaskId: "   "},
			code: codes.InvalidArgument,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.GetTask(ctxWithAuth("devtoken"), tc.req)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if status.Code(err) != tc.code {
				t.Fatalf("expected code %v, got %v", tc.code, status.Code(err))
			}
		})
	}
}

func TestTaskService_List_InvalidArgument(t *testing.T) {
	client, cleanup := newBufconnClient(t)
	defer cleanup()

	cases := []struct {
		name string
		req  *taskv1.ListTasksRequest
		code codes.Code
	}{
		{
			name: "non-numeric page_token",
			req:  &taskv1.ListTasksRequest{PageSize: 10, PageToken: "abc"},
			code: codes.InvalidArgument,
		},
		{
			name: "negative page_token",
			req:  &taskv1.ListTasksRequest{PageSize: 10, PageToken: "-1"},
			code: codes.InvalidArgument,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.ListTasks(ctxWithAuth("devtoken"), tc.req)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if status.Code(err) != tc.code {
				t.Fatalf("expected code %v, got %v", tc.code, status.Code(err))
			}
		})
	}
}

func TestTaskService_CreateWithId_InvalidArgument(t *testing.T) {
	client, cleanup := newBufconnClient(t)
	defer cleanup()
	cases := []struct {
		name string
		req  *taskv1.CreateTaskWithIdRequest
		code codes.Code
	}{
		{
			name: "empty task_id",
			req:  &taskv1.CreateTaskWithIdRequest{TaskId: "", Title: "Test Task"},
			code: codes.InvalidArgument,
		},
		{
			name: "whitespace task_id",
			req:  &taskv1.CreateTaskWithIdRequest{TaskId: "   ", Title: "Test Task"},
			code: codes.InvalidArgument,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.CreateTaskWithId(ctxWithAuth("devtoken"), tc.req)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if status.Code(err) != tc.code {
				t.Fatalf("expected code %v, got %v", tc.code, status.Code(err))
			}
		})
	}
}

func TestTaskService_CreateWithIdThenCreatewithId_AlreadyExists(t *testing.T) {
	client, cleanup := newBufconnClient(t)
	defer cleanup()
	_, err := client.CreateTaskWithId(ctxWithAuth("devtoken"), &taskv1.CreateTaskWithIdRequest{
		TaskId:      "task-123",
		Title:       "Test Task",
		Description: "This is a test task",
	})
	if err != nil {
		t.Fatalf("CreateTaskWithId failed: %v", err)
	}
	_, err = client.CreateTaskWithId(ctxWithAuth("devtoken"), &taskv1.CreateTaskWithIdRequest{
		TaskId:      "task-123",
		Title:       "Another Task",
		Description: "This task should fail",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error")
	}
	if st.Code() != codes.AlreadyExists {
		t.Fatalf("expected AlreadyExists, got %v", st.Code())
	}
}

func TestTaskService_CreateTask_Unauthenticated(t *testing.T) {
	client, cleanup := newBufconnClient(t)
	defer cleanup()
	_, err := client.CreateTask(context.Background(), &taskv1.CreateTaskRequest{
		Title:       "buy milk",
		Description: "tonight",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error")
	}
	if st.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", st.Code())
	}
}

func TestTaskService_CreateTask_PermissionDenied(t *testing.T) {
	client, cleanup := newBufconnClient(t)
	defer cleanup()
	_, err := client.CreateTask(ctxWithAuth("something"), &taskv1.CreateTaskRequest{
		Title:       "buy milk",
		Description: "tonight",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error")
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", st.Code())
	}
}

func TestTaskService_CreateTask_SimulatedFailure(t *testing.T) {
	client, server, cleanup := newBufconnClientWithServer(t)
	defer cleanup()
	server.FailNextUnavailable()
	_, err := client.CreateTask(ctxWithAuth("devtoken"), &taskv1.CreateTaskRequest{
		Title:       "buy milk",
		Description: "tonight",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error")
	}
	if st.Code() != codes.Unavailable {
		t.Fatalf("expected Unavailable, got %v", st.Code())
	}
}
func callWithRetry(ctx context.Context, attempts int, fn func(context.Context) error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn(ctx)
		if err == nil {
			return nil
		}
		if status.Code(err) == codes.Unavailable {
			continue
		} else {
			return err
		}
	}
	return err
}
func TestTaskService_CreateTask_RetrySucceed(t *testing.T) {
	client, server, cleanup := newBufconnClientWithServer(t)
	defer cleanup()
	server.FailNextUnavailable()
	err := callWithRetry(ctxWithAuth("devtoken"), 3, func(ctx context.Context) error {
		_, err := client.CreateTask(ctx, &taskv1.CreateTaskRequest{
			Title:       "buy milk",
			Description: "tonight",
		})
		return err
	})
	if err != nil {
		t.Fatalf("expected no error after retry, got %v", err)
	}
}

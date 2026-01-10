package retry

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CallWithRetry(ctx context.Context, attempts int, fn func(context.Context) error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn(ctx)
		if err == nil {
			return nil
		}
		if status.Code(err) == codes.Unavailable {
			time.Sleep(50 * time.Millisecond)
			continue
		} else {
			return err
		}
	}
	return err
}

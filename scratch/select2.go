package main

import (
	"context"
	"fmt"
	"time"
)

func main2() {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	select {
	case <-time.After(1 * time.Second):
		fmt.Println("work completed")
	case <-ctx.Done():
		fmt.Println("stopped:", ctx.Err())
	}
}

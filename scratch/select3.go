package main

import (
	"fmt"
	"time"
)

func main3() {
	ch := make(chan int)

	select {
	case v := <-ch:
		fmt.Println("received", v)
	default:
		fmt.Println("no value ready, moving on")
	}

	time.Sleep(200 * time.Millisecond)
	fmt.Println("done")
}

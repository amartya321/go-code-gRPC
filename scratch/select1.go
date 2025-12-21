package main

import (
	"fmt"
	"time"
)

// func main() {
// 	start := time.Now()

// 	select {
// 	case <-time.After(300 * time.Millisecond):
// 		fmt.Println("A fired at", time.Since(start))
// 	case <-time.After(100 * time.Millisecond):
// 		fmt.Println("B fired at", time.Since(start))
// 	}
// }

func main1() {
	start := time.Now()

	select {
	case <-time.After(300 * time.Millisecond):
		fmt.Println("A fired at", time.Since(start))
	case <-time.After(100 * time.Millisecond):
		fmt.Println("B fired at", time.Since(start))
	}
}

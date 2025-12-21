package main

import (
	"fmt"
	"math/rand"
	"time"
)

func producer(name string) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		for i := 1; i <= 5; i++ {
			time.Sleep(time.Duration(50+rand.Intn(150)) * time.Millisecond)
			ch <- fmt.Sprintf("%s: msg %d", name, i)
		}
	}()
	return ch
}

func main5() {
	rand.Seed(time.Now().UnixNano())

	a := producer("A")
	b := producer("B")

	// read 10 messages total, whichever arrives first
	for range 10 {
		select {
		case msg := <-a:
			fmt.Println(msg)
		case msg := <-b:
			fmt.Println(msg)
		}
	}
}

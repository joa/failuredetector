package main

import (
	"fmt"
	"time"

	"github.com/joa/failuredetector"
)

func main() {
	warnings := make(chan time.Duration)
	node, err := failuredetector.New(
		8.0,
		200,
		500*time.Millisecond,
		0,
		500*time.Millisecond,
		warnings)

	if err != nil {
		panic(err)
	}

	go func() {
		for {
			select {
			case d := <-warnings:
				fmt.Println("warning: heartbeat interval is growing too large", d)
			}
		}
	}()

	fmt.Println("node is alive")
	for i := 0; i < 5; i++ {
		node.Heartbeat()
		fmt.Printf("alive=%v\tphi=%.8f\n", node.IsAvailable(), node.Phi())
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("node crashed")
	for i := 0; i < 5; i++ {
		fmt.Printf("alive=%v\tphi=%.8f\n", node.IsAvailable(), node.Phi())
		time.Sleep(1 * time.Second)
	}

	fmt.Println("node is back up")
	for i := 0; i < 5; i++ {
		node.Heartbeat()
		fmt.Printf("alive=%v\tphi=%.8f\n", node.IsAvailable(), node.Phi())
		time.Sleep(500 * time.Millisecond)
	}

	close(warnings)
}

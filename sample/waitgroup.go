package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(4)
	go worker(&wg, "a", 4)
	go worker(&wg, "b", 3)
	go worker(&wg, "c", 2)
	go worker(&wg, "d", 1)
	fmt.Printf("main: waiting\n")
	wg.Wait()
	fmt.Printf("main: done\n")
}

func worker(wg *sync.WaitGroup, name string, sleep time.Duration) {
	fmt.Printf("worker %s: working\n", name)
	time.Sleep(sleep * time.Second)
	fmt.Printf("worker %s: done\n", name)
	wg.Done()
}

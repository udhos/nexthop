package main

import (
	"fmt"
	"sync"
)

type info struct {
	n      int
	read   chan int // channel for client to query information: DANGER client could block forever
	write  chan int // channel for client to update information: DANGER client could block forever
	quit   chan int // channel for client to request termination: DANGER client could perform multiple closes
	closed bool
	mutex  sync.RWMutex
}

// SafeClose: fixes multiple close
func (i *info) SafeClose() {
	defer i.mutex.Unlock()
	i.mutex.Lock()
	if i.closed {
		return
	}
	close(i.quit)
	i.closed = true
}

func newInfo(wg *sync.WaitGroup) *info {
	i := &info{read: make(chan int), write: make(chan int), quit: make(chan int)}

	go func() {
		fmt.Printf("info: running\n")

		defer wg.Done()

		for {
			select {
			case i.read <- i.n:
			case i.n = <-i.write:
			case <-i.quit:
				fmt.Printf("info: quitting\n")
				return
			}
		}
	}()

	return i
}

func main() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	i := newInfo(wg)

	v, _ := <-i.read
	fmt.Printf("main: read=%v\n", v)

	i.write <- 2

	v, _ = <-i.read
	fmt.Printf("main: read=%v\n", v)

	i.SafeClose()
	i.SafeClose()

	fmt.Printf("main: waiting termination\n")

	wg.Wait()

	fmt.Printf("main: end\n")
}

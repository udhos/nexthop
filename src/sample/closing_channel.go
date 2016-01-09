package main

import (
	"fmt"
)

func main() {

	output := make(chan int, 1) // create channel

	write(output, 1)

	close(output) // close channel

	write(output, 2)
}

// how to write on possibly closed channel
func write(out chan int, i int) {

	defer func() {
		// recover from panic caused by writing to a closed channel
		if err := recover(); err != nil {
			fmt.Printf("write: error writing %d on channel: %v\n", i, err)
			return
		}

		fmt.Printf("write: wrote %d on channel\n", i)
	}()

	out <- i // write on possibly closed channel
}

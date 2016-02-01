package main

type RipRouter struct {
	done chan int // request end of rip router
}

func NewRipRouter() *RipRouter {
	return &RipRouter{done: make(chan int)}
}

func (r *RipRouter) NetAdd(net string) {
}

func (r *RipRouter) NetDel(net string) {
}

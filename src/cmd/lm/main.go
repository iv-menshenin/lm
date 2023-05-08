package main

import (
	"github.com/iv-menshenin/lm/coordinator"
	"github.com/iv-menshenin/lm/transport"
)

func main() {
	udp, err := transport.NewUDP(7999)
	if err != nil {
		panic(err)
	}
	c := coordinator.New(udp)
	if err := c.Manage(); err != nil {
		panic(err)
	}
}

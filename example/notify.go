package main

import (
	"log"
	"github.com/shopex/teegon-open-go-sdk"
)

func main() {
	c, err := teegon.NewClient("ws://api.teegon.com/router", "4234", "4242342423434")
	if err != nil {
		log.Println("create client: ", err)
	}

	n, err := c.Notify()
	if err != nil {
		log.Println("create notify: ", err)
		return
	}

	ch, err := n.Consume("sendmsg")
	if err != nil {
		log.Println("consume queue: ", err)
	}

	for {
		msg := <-ch
		log.Printf("%#v\n", msg)
		msg.Ack()
	}
}

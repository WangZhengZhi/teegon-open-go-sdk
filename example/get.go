package main

import (
	"log"
	"github.com/shopex/teegon-open-go-sdk"
	"fmt"
)

func main() {
	c, err := teegon.NewClient("http://api.teegon.com/router", "32424", "43244")
	if err != nil {
		log.Println("create client: ", err)
	}

	r ,err := c.Get("shopex.query.appqueue", &map[string]interface{}{
		"user_eid":"321312312",
		"app_id": "test1",
	})
	fmt.Printf(string(r.Raw))
}

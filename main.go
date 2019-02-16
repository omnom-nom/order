package main

import (
	"time"

	"github.com/omnom-nom/order/api"
)

func main() {

	err := api.Init()
	if err != nil {
		return
	}

	for {
                time.Sleep(120 * time.Second)
                continue
        }
}

package main

import (
	"log"

	"github.com/atomicptr/sidewinder/pkg/cli"
)

func main() {
	err := cli.Run()
	if err != nil {
		log.Fatal(err)
	}
}

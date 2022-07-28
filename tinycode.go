package main

import (
	"log"
	"github.com/brokad/tinycode/cmd"
)

func main() {
	log.SetPrefix("tinycode: ")
	log.SetFlags(0)
	cmd.Execute()
}

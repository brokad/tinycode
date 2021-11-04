package main

import (
	"github.com/brokad/tinycode/cmd"
	"log"
)

func main() {
	log.SetPrefix("tinycode: ")
	log.SetFlags(0)
	cmd.Execute()
}
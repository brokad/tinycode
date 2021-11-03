package main

import (
	"github.com/brokad/tinycode/cmd"
	"log"
	"os"
)

func main() {
	log.SetPrefix("tinycode: ")
	log.SetOutput(os.Stderr)
	log.SetFlags(0)

	cmd.Execute()
}
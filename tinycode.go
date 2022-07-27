package main

import (
	"log"
	"tinycode/cmd"
)

func main() {
	log.SetPrefix("tinycode: ")
	log.SetFlags(0)
	cmd.Execute()
}

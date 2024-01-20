package main

import (
	"log"
	"strings"
)

const (
	progName = "rgap"
)

var (
	version = "undefined"
)

func main() {
	log.Default().SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Default().SetPrefix(strings.ToUpper(progName) + ": ")
	Execute()
}

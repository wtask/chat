package main

import (
	stdlog "log"
	"os"
)

type (
	// Logger - logging interface used by the server
	Logger interface {
		Println(v ...interface{})
		Printf(format string, v ...interface{})
	}
)

func main() {
	var log Logger = stdlog.New(os.Stdout, BinaryName+":"+Version+" ", stdlog.Ldate|stdlog.Ltime)
	log.Printf("Starting with config: %+v", Config)
}

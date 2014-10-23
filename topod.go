package main

import (
	"fmt"
	"github.com/op/go-logging"
	"os"
)

var log = logging.MustGetLogger("topod")
var format = "%{color}%{time:2006-01-02 15:04:05.000000} > %{level:.3s} %{id:03x}%{color:reset} %{message}"

func init() {
	logBackend := logging.NewLogBackend(os.Stdout, "", 0)
	logging.SetBackend(logBackend)
	logging.SetFormatter(logging.MustStringFormatter(format))
	logging.SetLevel(logging.DEBUG, "topod")
}

func main() {
	//flag.Parse()
	if options.Version {
		fmt.Printf("Topod version %s\n", Version)
		os.Exit(0)
	}
}

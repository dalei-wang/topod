package logger

import (
	"os"

	"github.com/op/go-logging"
)

var Log = logging.MustGetLogger("topod")
var format = "%{color}%{time:2006-01-02 15:04:05.000000} > %{level:.3s} %{id:03x}%{color:reset} %{message}"

func init() {
	logBackend := logging.NewLogBackend(os.Stdout, "", 0)
	logging.SetBackend(logBackend)
	logging.SetFormatter(logging.MustStringFormatter(format))
	//logging.SetLevel(logging.DEBUG, "topod")
}

func SetLevel(isDebug, isVerbose bool) {
	if isDebug {
		logging.SetLevel(logging.DEBUG, "topod")
	} else if isVerbose {
		logging.SetLevel(logging.NOTICE, "topod")
	} else {
		logging.SetLevel(logging.INFO, "topod")
	}
}

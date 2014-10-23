package store

import (
	"errors"
	"github.com/op/go-logging"
	"os"

	"github.com/wlsailor/topod/store/etcd"
)

var log = logging.MustGetLogger("store")
var format = "%{color}%{time:2006-01-02 15:04:05.000000} > %{level:.3s} %{id:03x}%{color:reset} %{message}"

func init() {
	logBackend := logging.NewLogBackend(os.Stdout, "", 0)
	logging.SetBackend(logBackend)
	logging.SetFormatter(logging.MustStringFormatter(format))
	logging.SetLevel(logging.DEBUG, "store")
}

type StoreClient interface {
	GetValues(keys []string) (map[string]string, error)
	WatchPrefix(prefix string, waitIndex uint64, stopChan chan bool) (uint64, error)
}

func NewClient(config Config) (StoreClient, error) {
	if config.Store == "" {
		config.Store = "etcd"
	}
	storeNodes := config.Nodes
	log.Notice("Store nodes set to %v", storeNodes)
	switch config.Store {
	case "etcd":
		return etcd.NewClient(storeNodes, config.Cert, config.Key, config.CaKeys)
	}
	return nil, errors.New("Invalid store config")
}

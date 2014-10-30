package store

import (
	"errors"

	"github.com/wlsailor/topod/logger"
	"github.com/wlsailor/topod/store/etcd"
)

type StoreClient interface {
	GetValues(keys []string) (map[string]string, error)
	WatchPrefix(prefix string, waitIndex uint64, stopChan chan bool) (uint64, error)
}

func NewClient(config Config) (StoreClient, error) {
	if config.Store == "" {
		config.Store = "etcd"
	}
	storeNodes := config.Nodes
	logger.Log.Notice("Store nodes set to %v", storeNodes)
	switch config.Store {
	case "etcd":
		return etcd.NewClient(storeNodes, config.Cert, config.Key, config.CaKeys)
	}
	return nil, errors.New("Invalid store config")
}

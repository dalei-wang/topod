package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wlsailor/topod/conf/template"
	"github.com/wlsailor/topod/logger"
	storage "github.com/wlsailor/topod/store"
)

func main() {
	//flag.Parse()
	logger.SetLevel(options.Debug, options.Verbose)
	if options.Version {
		fmt.Printf("Topod version %s\n", Version)
		os.Exit(0)
	}
	if err := initConfig(); err != nil {
		logger.Log.Fatal(err.Error())
	}
	logger.Log.Notice("Starting topod")
	storeClient, _ := storage.NewClient(storeConfig)
	templateConfig.StoreClient = storeClient
	//values, _ := storeClient.GetValues([]string{"/products/kmsocial/mysql/url"})
	//value := values["/products/kmsocial/mysql/url"]
	//fmt.Println("Value of key /products/kmsocial/mysql/url", value)
	if options.Verbs == "gen" {
		if err := template.ProcessOnce(&templateConfig); err != nil {
			logger.Log.Error("Generate config file error: %s", err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}

	stopChan := make(chan bool)
	doneChan := make(chan bool)
	errChan := make(chan error, 10)
	var processor template.Processor
	switch options.Verbs {
	case "watch":
		processor = template.NewWatcher(&templateConfig, stopChan, doneChan, errChan)
	case "pull":
		fmt.Println("Pull interval not implement")
		os.Exit(0)
	default:
		processor = template.NewWatcher(&templateConfig, stopChan, doneChan, errChan)
	}
	go processor.Process()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case err := <-errChan:
			logger.Log.Error(err.Error())
		case s := <-signalChan:
			logger.Log.Info("captured %v exiting...", s)
			close(doneChan)
		case <-doneChan:
			os.Exit(0)
		}
	}
}

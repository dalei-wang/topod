package template

import (
	"sync"

	"github.com/wlsailor/topod/logger"
)

type Watcher struct {
	config   *Config
	stopChan chan bool
	doneChan chan bool
	errChan  chan error
	wg       *sync.WaitGroup
}

func NewWatcher(config *Config, stopChan, doneChan chan bool, errChan chan error) Processor {
	var wg sync.WaitGroup
	return &Watcher{
		config, stopChan, doneChan, errChan, &wg,
	}
}

func (w *Watcher) Process() {
	defer close(w.doneChan)
	ts, err := getTemplateResource(w.config)
	if err != nil {
		logger.Log.Error("Get template resource error: %s", err.Error())
		return
	}
	for _, t := range ts {
		w.wg.Add(1)
		go w.monitorPrefix(t)
	}
	w.wg.Wait()
}

func (p *Watcher) monitorPrefix(t *TemplateResource) {
	defer p.wg.Done()
	for {
		logger.Log.Debug("Begin watching prefix %s with index %d", t.Prefix, t.lastIndex)
		index, err := p.config.StoreClient.WatchPrefix(t.Prefix, t.lastIndex, p.stopChan)
		if err != nil {
			logger.Log.Error("Watching prefix key %s error: %s", t.Prefix, err.Error())
			p.errChan <- err
			continue
		}
		logger.Log.Debug("Watching prefix key %s changed, ready to process", t.Prefix)
		t.lastIndex = index
		if err := t.process(); err != nil {
			p.errChan <- err
		}
	}
}

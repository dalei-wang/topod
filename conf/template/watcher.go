package template

import (
	"sync"
)

type Watcher struct {
	config   Config
	stopChan chan bool
	doneChan chan bool
	errChan  chan error
	wg       *sync.WaitGroup
}

func newWatcher(config Config, stopChan, doneChan bool, errChan chan error) Processor {
	var wg sync.WaitGroup
	return &Watcher{
		config, stopChan, doneChan, errChan, &wg,
	}
}

func (w *Watcher) Process() {
	defer close(w.doneChan)
	ts, err := getTemplateResource(w.config)
	if err != nil {
		log.Error("Get template resource error: %s", err.Error())
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
		index, err := p.config.StoreClient.WatchPrefix(t.Prefix, t.lastIndex, p.stopChan)
		if err != nil {
			p.errChan <- err
			continue
		}
		t.lastIndex = index
		if err := t.process(); err != nil {
			p.errChan <- err
		}
	}
}

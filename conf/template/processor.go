package template

import (
	"github.com/wlsailor/topod/logger"
)

type Processor interface {
	Process()
}

func ProcessOnce(config *Config) error {
	templates, err := getTemplateResource(config)
	if err != nil {
		return err
	}
	var lastError error
	for _, t := range templates {
		if err := t.process(); err != nil {
			logger.Log.Error("Process template source %s error: %s", t.Src, err.Error())
			lastError = err
		}
	}
	logger.Log.Info("Process all template source done")
	return lastError
}

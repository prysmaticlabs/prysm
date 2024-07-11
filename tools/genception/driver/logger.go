package driver

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// Create a new instance of the logger. You can have any number of instances.
var log = logrus.New()
var Logger *logrus.Logger

func init() {
	path := os.Getenv("GOPACKAGESDRIVER_LOG_PATH")
	if path == "" {
		path = filepath.Join(os.Getenv("PWD"), "genception.log")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.Out = file
	} else {
		log.Info("Failed to log to file, using default stderr")
	}
	Logger = log
}

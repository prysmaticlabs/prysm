// Package logutil creates a Multi writer instance that
// write all logs that are written to stdout.
package logutil

import (
	"io"
	"os"
	"time"
	"github.com/sirupsen/logrus"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

var log = logrus.WithField("prefix", "logutil")

// ConfigurePersistentLogging adds a log-to-file writer. File content is identical to stdout.
func ConfigurePersistentLogging(logFileName string) error {
	logrus.WithField("logFileName", logFileName).Info("Logs will be made persistent")
	f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	mw := io.MultiWriter(os.Stdout, f)
	logrus.SetOutput(mw)

	logrus.Info("File logging initialized")
	return nil
}

// CountdownToGenesis logs the time remaining until the specified genesis time.
func CountdownToGenesis(genesisTime time.Time, duration time.Duration) {
	ticker := time.NewTicker(duration * time.Second)

	for {
		select {
		case <-time.NewTimer(genesisTime.Sub(roughtime.Now()) + 1 /* adding one to stop after the last minute */).C:
			return

		case <-ticker.C:
			log.Infof("%s to genesis.", genesisTime.Sub(roughtime.Now()) * time.Minute)
		}
	}
} 

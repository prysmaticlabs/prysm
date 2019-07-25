// Package logutil creates a Multi writer instance that
// write all logs that are written to stdout.
package logutil

import (
	"io"
	"os"
	"reflect"
	"runtime"

	"github.com/sirupsen/logrus"
)

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

// FunctionName returns the string representation of the function name for some interface.
func FunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

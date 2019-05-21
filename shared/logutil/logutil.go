// Package logutil creates a logrus lib file logger instance that
// write all logs that are written to stdout.
package logutil

import (
	"fmt"
	"os"
	"strings"

	joonix "github.com/joonix/log"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var _ = logrus.Hook(&WriterHook{})

// WriterHook is a hook that writes logs of specified LogLevels to specified Writer.
type WriterHook struct {
	LogLevels []logrus.Level
}

// Fire will be called when some logging function is called with current hook.
// It will format log entry to string and write it to appropriate writer.
func (hook *WriterHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}
	//simply call the file logger Println func after removing the new line char
	line = strings.TrimSuffix(line, "\n")
	fileLogger.Println(line)
	return err
}

// Levels defines on which log levels this hook would trigger.
func (hook *WriterHook) Levels() []logrus.Level {
	return hook.LogLevels
}

var fileLogger = &logrus.Logger{
	Level: logrus.TraceLevel,
}

// ConfigurePersistentLogging adds a log-to-file writer hook to the logrus logger. The writer hook appends new
// logs to the specified log file.
func ConfigurePersistentLogging(logFileName string, logFileFormatName string) error {
	logrus.WithField("logFileName", logFileName).Info("Logs will be made persistent")
	f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	fileLogger.SetOutput(f)

	switch logFileFormatName {
	case "text":
		formatter := new(prefixed.TextFormatter)
		formatter.TimestampFormat = "2006-01-02 15:04:05"
		formatter.FullTimestamp = true
		formatter.DisableColors = true
		fileLogger.SetFormatter(formatter)
		break
	case "fluentd":
		fileLogger.SetFormatter(&joonix.FluentdFormatter{})
		break
	case "json":
		fileLogger.SetFormatter(&logrus.JSONFormatter{})
		break
	default:
		return fmt.Errorf("unknown log file format %v", logFileFormatName)
	}

	logrus.Info("File logger initialized")
	logrus.AddHook(&WriterHook{
		LogLevels: logrus.AllLevels,
	})

	return nil
}

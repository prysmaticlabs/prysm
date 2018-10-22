package log

import (
	"errors"
	"fmt"
	"os"
	"sync"

	tracer "github.com/ipfs/go-log/tracer"

	colorable "github.com/mattn/go-colorable"
	opentrace "github.com/opentracing/opentracing-go"
	logging "github.com/whyrusleeping/go-logging"
)

func init() {
	SetupLogging()
}

var ansiGray = "\033[0;37m"
var ansiBlue = "\033[0;34m"

// LogFormats defines formats for logging (i.e. "color")
var LogFormats = map[string]string{
	"nocolor": "%{time:2006-01-02 15:04:05.000000} %{level} %{module} %{shortfile}: %{message}",
	"color": ansiGray + "%{time:15:04:05.000} %{color}%{level:5.5s} " + ansiBlue +
		"%{module:10.10s}: %{color:reset}%{message} " + ansiGray + "%{shortfile}%{color:reset}",
}

var defaultLogFormat = "color"

// Logging environment variables
const (
	// TODO these env names should be more general, IPFS is not the only project to
	// use go-log
	envLogging    = "IPFS_LOGGING"
	envLoggingFmt = "IPFS_LOGGING_FMT"

	envLoggingFile = "GOLOG_FILE" // /path/to/file
)

// ErrNoSuchLogger is returned when the util pkg is asked for a non existant logger
var ErrNoSuchLogger = errors.New("Error: No such logger")

// loggers is the set of loggers in the system
var loggerMutex sync.RWMutex
var loggers = map[string]*logging.Logger{}

// SetupLogging will initialize the logger backend and set the flags.
// TODO calling this in `init` pushes all configuration to env variables
// - move it out of `init`? then we need to change all the code (js-ipfs, go-ipfs) to call this explicitly
// - have it look for a config file? need to define what that is
func SetupLogging() {

	// colorful or plain
	lfmt := LogFormats[os.Getenv(envLoggingFmt)]
	if lfmt == "" {
		lfmt = LogFormats[defaultLogFormat]
	}

	// check if we log to a file
	var lgbe []logging.Backend
	if logfp := os.Getenv(envLoggingFile); len(logfp) > 0 {
		f, err := os.Create(logfp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR go-log: %s: failed to set logging file backend\n", err)
		} else {
			lgbe = append(lgbe, logging.NewLogBackend(f, "", 0))
		}
	}

	// logs written to stderr
	lgbe = append(lgbe, logging.NewLogBackend(colorable.NewColorableStderr(), "", 0))

	// set the backend(s)
	logging.SetBackend(lgbe...)
	logging.SetFormatter(logging.MustStringFormatter(lfmt))

	lvl := logging.ERROR

	if logenv := os.Getenv(envLogging); logenv != "" {
		var err error
		lvl, err = logging.LogLevel(logenv)
		if err != nil {
			fmt.Println("error setting log levels", err)
		}
	}

	// TracerPlugins are instantiated after this, so use loggable tracer
	// by default, if a TracerPlugin is added it will override this
	lgblRecorder := tracer.NewLoggableRecorder()
	lgblTracer := tracer.New(lgblRecorder)
	opentrace.SetGlobalTracer(lgblTracer)

	SetAllLoggers(lvl)
}

// SetDebugLogging calls SetAllLoggers with logging.DEBUG
func SetDebugLogging() {
	SetAllLoggers(logging.DEBUG)
}

// SetAllLoggers changes the logging.Level of all loggers to lvl
func SetAllLoggers(lvl logging.Level) {
	logging.SetLevel(lvl, "")

	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	for n := range loggers {
		logging.SetLevel(lvl, n)
	}
}

// SetLogLevel changes the log level of a specific subsystem
// name=="*" changes all subsystems
func SetLogLevel(name, level string) error {
	lvl, err := logging.LogLevel(level)
	if err != nil {
		return err
	}

	// wildcard, change all
	if name == "*" {
		SetAllLoggers(lvl)
		return nil
	}

	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	// Check if we have a logger by that name
	if _, ok := loggers[name]; !ok {
		return ErrNoSuchLogger
	}

	logging.SetLevel(lvl, name)

	return nil
}

// GetSubsystems returns a slice containing the
// names of the current loggers
func GetSubsystems() []string {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	subs := make([]string, 0, len(loggers))

	for k := range loggers {
		subs = append(subs, k)
	}
	return subs
}

func getLogger(name string) *logging.Logger {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	log := loggers[name]
	if log == nil {
		log = logging.MustGetLogger(name)
		loggers[name] = log
	}

	return log
}

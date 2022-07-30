package testdata

import "errors"

var log = &logger{}

func LogThis(err error) {
	// Most common use cases with all types of formatted log methods.
	log.Debugf("Something really bad happened: %v", err)   // want "use log.WithError rather than templated log statements with errors"
	log.Infof("Something really bad happened: %v", err)    // want "use log.WithError rather than templated log statements with errors"
	log.Printf("Something really bad happened: %v", err)   // want "use log.WithError rather than templated log statements with errors"
	log.Warnf("Something really bad happened: %v", err)    // want "use log.WithError rather than templated log statements with errors"
	log.Warningf("Something really bad happened: %v", err) // want "use log.WithError rather than templated log statements with errors"
	log.Errorf("Something really bad happened: %v", err)   // want "use log.WithError rather than templated log statements with errors"
	log.Fatalf("Something really bad happened: %v", err)   // want "use log.WithError rather than templated log statements with errors"
	log.Panicf("Something really bad happened: %v", err)   // want "use log.WithError rather than templated log statements with errors"

	// Inline declaration of errors and multiple value arguments.
	log.Panicf("Something really bad happened: %v", errors.New("foobar")) // want "use log.WithError rather than templated log statements with errors"
	log.Panicf("Something really bad happened %d times: %v", 12, err)     // want "use log.WithError rather than templated log statements with errors"

	log.WithError(err).Error("Something bad happened, but this log statement is OK :)")
}

type logger struct{}

func (*logger) Debugf(format string, args ...interface{}) {
}

func (*logger) Infof(format string, args ...interface{}) {
}

func (*logger) Printf(format string, args ...interface{}) {
}

func (*logger) Warnf(format string, args ...interface{}) {
}

func (*logger) Warningf(format string, args ...interface{}) {
}

func (*logger) Errorf(format string, args ...interface{}) {
}

func (*logger) Error(msg string) {
}

func (*logger) Fatalf(format string, args ...interface{}) {
}

func (*logger) Panicf(format string, args ...interface{}) {
}

func (l *logger) WithError(err error) *logger {
	return l
}

package testdata

var log = &logger{}

func LogThis(err error) {
	log.Debugf("Something really bad happened: %v", err)   // want "use logrus.WithError(err) rather than templated log statements"
	log.Infof("Something really bad happened: %v", err)    // want "use logrus.WithError(err) rather than templated log statements"
	log.Printf("Something really bad happened: %v", err)   // want "use logrus.WithError(err) rather than templated log statements"
	log.Warnf("Something really bad happened: %v", err)    // want "use logrus.WithError(err) rather than templated log statements"
	log.Warningf("Something really bad happened: %v", err) // want "use logrus.WithError(err) rather than templated log statements"
	log.Errorf("Something really bad happened: %v", err)   // want "use logrus.WithError(err) rather than templated log statements"
	log.Fatalf("Something really bad happened: %v", err)   // want "use logrus.WithError(err) rather than templated log statements"
	log.Panicf("Something really bad happened: %v", err)   // want "use logrus.WithError(err) rather than templated log statements"

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

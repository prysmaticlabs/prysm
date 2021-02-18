package sszutil

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/d4l3k/messagediff"
	"github.com/prysmaticlabs/prysm/shared/testutil/assertions"
)

type assertionLoggerFn func(string, ...interface{})

func AssertDeepEqual(tb assertions.AssertionTestingTB, expected, actual interface{}, msg ...interface{}) {
	deepSSZEqual(tb.Errorf, expected, actual, msg...)
}

func AssertDeepNotEqual(tb assertions.AssertionTestingTB, expected, actual interface{}, msg ...interface{}) {
	deepNotSSZEqual(tb.Errorf, expected, actual, msg...)
}

func RequireDeepEqual(tb assertions.AssertionTestingTB, expected, actual interface{}, msg ...interface{}) {
	deepSSZEqual(tb.Fatalf, expected, actual, msg...)
}

func RequireDeepNotEqual(tb assertions.AssertionTestingTB, expected, actual interface{}, msg ...interface{}) {
	deepNotSSZEqual(tb.Fatalf, expected, actual, msg...)
}

func deepSSZEqual(loggerFn assertionLoggerFn, expected, actual interface{}, msg ...interface{}) {
	if !DeepEqual(expected, actual) {
		errMsg := parseMsg("Values are not equal", msg...)
		_, file, line, _ := runtime.Caller(2)
		diff, _ := messagediff.PrettyDiff(expected, actual)
		loggerFn("%s:%d %s, want: %#v, got: %#v, diff: %s", filepath.Base(file), line, errMsg, expected, actual, diff)
	}
}

func deepNotSSZEqual(loggerFn assertionLoggerFn, expected, actual interface{}, msg ...interface{}) {
	if DeepEqual(expected, actual) {
		errMsg := parseMsg("Values are equal", msg...)
		_, file, line, _ := runtime.Caller(2)
		loggerFn("%s:%d %s, want: %#v, got: %#v", filepath.Base(file), line, errMsg, expected, actual)
	}
}

func parseMsg(defaultMsg string, msg ...interface{}) string {
	if len(msg) >= 1 {
		msgFormat, ok := msg[0].(string)
		if !ok {
			return defaultMsg
		}
		return fmt.Sprintf(msgFormat, msg[1:]...)
	}
	return defaultMsg
}

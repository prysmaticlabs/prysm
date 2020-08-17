package assert

import (
	"github.com/prysmaticlabs/prysm/shared/testutil/assertions"
	"github.com/sirupsen/logrus/hooks/test"
)

// Equal compares values using comparison operator.
func Equal(tb assertions.AssertionTestingTB, expected, actual interface{}, msg ...interface{}) {
	assertions.Equal(tb.Errorf, expected, actual, msg...)
}

// NotEqual compares values using comparison operator.
func NotEqual(tb assertions.AssertionTestingTB, expected, actual interface{}, msg ...interface{}) {
	assertions.NotEqual(tb.Errorf, expected, actual, msg...)
}

// DeepEqual compares values using DeepEqual.
func DeepEqual(tb assertions.AssertionTestingTB, expected, actual interface{}, msg ...interface{}) {
	assertions.DeepEqual(tb.Errorf, expected, actual, msg...)
}

// NoError asserts that error is nil.
func NoError(tb assertions.AssertionTestingTB, err error, msg ...interface{}) {
	assertions.NoError(tb.Errorf, err, msg...)
}

// ErrorContains asserts that actual error contains wanted message.
func ErrorContains(tb assertions.AssertionTestingTB, want string, err error, msg ...interface{}) {
	assertions.ErrorContains(tb.Errorf, want, err, msg...)
}

// NotNil asserts that passed value is not nil.
func NotNil(tb assertions.AssertionTestingTB, obj interface{}, msg ...interface{}) {
	assertions.NotNil(tb.Errorf, obj, msg...)
}

// LogsContain checks that the desired string is a subset of the current log output.
func LogsContain(tb assertions.AssertionTestingTB, hook *test.Hook, want string, msg ...interface{}) {
	assertions.LogsContain(tb.Errorf, hook, want, true, msg...)
}

// LogsDoNotContain is the inverse check of LogsContain.
func LogsDoNotContain(tb assertions.AssertionTestingTB, hook *test.Hook, want string, msg ...interface{}) {
	assertions.LogsContain(tb.Errorf, hook, want, false, msg...)
}

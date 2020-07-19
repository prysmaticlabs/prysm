package require

import (
	"github.com/prysmaticlabs/prysm/shared/testutil/assertions"
)

// Equal compares values using comparison operator.
func Equal(tb assertions.AssertionTestingTB, expected, actual interface{}, msg ...string) {
	assertions.Equal(tb.Fatalf, expected, actual, msg...)
}

// DeepEqual compares values using DeepEqual.
func DeepEqual(tb assertions.AssertionTestingTB, expected, actual interface{}, msg ...string) {
	assertions.DeepEqual(tb.Fatalf, expected, actual, msg...)
}

// NoError asserts that error is nil.
func NoError(tb assertions.AssertionTestingTB, err error, msg ...string) {
	assertions.NoError(tb.Fatalf, err, msg...)
}

// ErrorContains asserts that actual error contains wanted message.
func ErrorContains(tb assertions.AssertionTestingTB, want string, err error, msg ...string) {
	assertions.ErrorContains(tb.Fatalf, want, err, msg...)
}

// NotNil asserts that passed value is not nil.
func NotNil(tb assertions.AssertionTestingTB, obj interface{}, msg ...string) {
	assertions.NotNil(tb.Fatalf, obj, msg...)
}

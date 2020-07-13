package require

import (
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// Equal compares values using comparison operator.
func Equal(tb testutil.AssertionTestingTB, expected, actual interface{}, msg ...string) {
	testutil.Equal(tb.Fatalf, expected, actual, msg...)
}

// DeepEqual compares values using DeepEqual.
func DeepEqual(tb testutil.AssertionTestingTB, expected, actual interface{}, msg ...string) {
	testutil.DeepEqual(tb.Fatalf, expected, actual, msg...)
}

// NoError asserts that error is nil.
func NoError(tb testutil.AssertionTestingTB, err error, msg ...string) {
	testutil.NoError(tb.Fatalf, err, msg...)
}

// ErrorContains asserts that actual error contains wanted message.
func ErrorContains(tb testutil.AssertionTestingTB, want string, err error, msg ...string) {
	testutil.ErrorContains(tb.Fatalf, want, err, msg...)
}

// NotNil asserts that passed value is not nil.
func NotNil(tb testutil.AssertionTestingTB, obj interface{}, msg ...string) {
	testutil.NotNil(tb.Fatalf, obj, msg...)
}

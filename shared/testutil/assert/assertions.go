package assert

import (
	"github.com/prysmaticlabs/prysm/shared/testutil/assertions"
)

// Equal compares values using comparison operator.
func Equal(tb assertions.AssertionTestingTB, x, y interface{}, msg ...string) {
	assertions.Equal(tb.Errorf, x, y, msg...)
}

// DeepEqual compares values using DeepEqual.
func DeepEqual(tb assertions.AssertionTestingTB, x, y interface{}, msg ...string) {
	assertions.DeepEqual(tb.Errorf, x, y, msg...)
}

// NoError asserts that error is nil.
func NoError(tb assertions.AssertionTestingTB, err error, msg ...string) {
	assertions.NoError(tb.Errorf, err, msg...)
}

// ErrorContains asserts that actual error contains wanted message.
func ErrorContains(tb assertions.AssertionTestingTB, want string, err error, msg ...string) {
	assertions.ErrorContains(tb.Errorf, want, err, msg...)
}

// NotNil asserts that passed value is not nil.
func NotNil(tb assertions.AssertionTestingTB, obj interface{}, msg ...string) {
	assertions.NotNil(tb.Errorf, obj, msg...)
}

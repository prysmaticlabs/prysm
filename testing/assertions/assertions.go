package assertions

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/d4l3k/messagediff"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	"github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/protobuf/proto"
)

// AssertionTestingTB exposes enough testing.TB methods for assertions.
type AssertionTestingTB interface {
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

type assertionLoggerFn func(string, ...interface{})

// Equal compares values using comparison operator.
func Equal(loggerFn assertionLoggerFn, expected, actual interface{}, msg ...interface{}) {
	if expected != actual {
		errMsg := parseMsg("Values are not equal", msg...)
		_, file, line, _ := runtime.Caller(2)
		loggerFn("%s:%d %s, want: %[4]v (%[4]T), got: %[5]v (%[5]T)", filepath.Base(file), line, errMsg, expected, actual)
	}
}

// NotEqual compares values using comparison operator.
func NotEqual(loggerFn assertionLoggerFn, expected, actual interface{}, msg ...interface{}) {
	if expected == actual {
		errMsg := parseMsg("Values are equal", msg...)
		_, file, line, _ := runtime.Caller(2)
		loggerFn("%s:%d %s, both values are equal: %[4]v (%[4]T)", filepath.Base(file), line, errMsg, expected)
	}
}

// DeepEqual compares values using DeepEqual.
func DeepEqual(loggerFn assertionLoggerFn, expected, actual interface{}, msg ...interface{}) {
	if !isDeepEqual(expected, actual) {
		errMsg := parseMsg("Values are not equal", msg...)
		_, file, line, _ := runtime.Caller(2)
		diff, _ := messagediff.PrettyDiff(expected, actual)
		loggerFn("%s:%d %s, want: %#v, got: %#v, diff: %s", filepath.Base(file), line, errMsg, expected, actual, diff)
	}
}

// DeepNotEqual compares values using DeepEqual.
func DeepNotEqual(loggerFn assertionLoggerFn, expected, actual interface{}, msg ...interface{}) {
	if isDeepEqual(expected, actual) {
		errMsg := parseMsg("Values are equal", msg...)
		_, file, line, _ := runtime.Caller(2)
		loggerFn("%s:%d %s, want: %#v, got: %#v", filepath.Base(file), line, errMsg, expected, actual)
	}
}

// DeepSSZEqual compares values using ssz.DeepEqual.
func DeepSSZEqual(loggerFn assertionLoggerFn, expected, actual interface{}, msg ...interface{}) {
	if !ssz.DeepEqual(expected, actual) {
		errMsg := parseMsg("Values are not equal", msg...)
		_, file, line, _ := runtime.Caller(2)
		diff, _ := messagediff.PrettyDiff(expected, actual)
		loggerFn("%s:%d %s, want: %#v, got: %#v, diff: %s", filepath.Base(file), line, errMsg, expected, actual, diff)
	}
}

// DeepNotSSZEqual compares values using ssz.DeepEqual.
func DeepNotSSZEqual(loggerFn assertionLoggerFn, expected, actual interface{}, msg ...interface{}) {
	if ssz.DeepEqual(expected, actual) {
		errMsg := parseMsg("Values are equal", msg...)
		_, file, line, _ := runtime.Caller(2)
		loggerFn("%s:%d %s, want: %#v, got: %#v", filepath.Base(file), line, errMsg, expected, actual)
	}
}

// NoError asserts that error is nil.
func NoError(loggerFn assertionLoggerFn, err error, msg ...interface{}) {
	if err != nil {
		errMsg := parseMsg("Unexpected error", msg...)
		_, file, line, _ := runtime.Caller(2)
		loggerFn("%s:%d %s: %v", filepath.Base(file), line, errMsg, err)
	}
}

// ErrorIs uses Errors.Is to recursively unwrap err looking for target in the chain.
// If any error in the chain matches target, the assertion will pass.
func ErrorIs(loggerFn assertionLoggerFn, err, target error, msg ...interface{}) {
	if !errors.Is(err, target) {
		errMsg := parseMsg(fmt.Sprintf("error %s not in chain", target), msg...)
		_, file, line, _ := runtime.Caller(2)
		loggerFn("%s:%d %s: %v", filepath.Base(file), line, errMsg, err)
	}
}

// ErrorContains asserts that actual error contains wanted message.
func ErrorContains(loggerFn assertionLoggerFn, want string, err error, msg ...interface{}) {
	if err == nil || !strings.Contains(err.Error(), want) {
		errMsg := parseMsg("Expected error not returned", msg...)
		_, file, line, _ := runtime.Caller(2)
		loggerFn("%s:%d %s, got: %v, want: %s", filepath.Base(file), line, errMsg, err, want)
	}
}

// NotNil asserts that passed value is not nil.
func NotNil(loggerFn assertionLoggerFn, obj interface{}, msg ...interface{}) {
	if isNil(obj) {
		errMsg := parseMsg("Unexpected nil value", msg...)
		_, file, line, _ := runtime.Caller(2)
		loggerFn("%s:%d %s", filepath.Base(file), line, errMsg)
	}
}

// isNil checks that underlying value of obj is nil.
func isNil(obj interface{}) bool {
	if obj == nil {
		return true
	}
	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return value.IsNil()
	}
	return false
}

// LogsContain checks whether a given substring is a part of logs. If flag=false, inverse is checked.
func LogsContain(loggerFn assertionLoggerFn, hook *test.Hook, want string, flag bool, msg ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	entries := hook.AllEntries()
	var logs []string
	match := false
	for _, e := range entries {
		msg, err := e.String()
		if err != nil {
			loggerFn("%s:%d Failed to format log entry to string: %v", filepath.Base(file), line, err)
			return
		}
		if strings.Contains(msg, want) {
			match = true
		}
		for _, field := range e.Data {
			fieldStr, ok := field.(string)
			if !ok {
				continue
			}
			if strings.Contains(fieldStr, want) {
				match = true
			}
		}
		logs = append(logs, msg)
	}
	var errMsg string
	if flag && !match {
		errMsg = parseMsg("Expected log not found", msg...)
	} else if !flag && match {
		errMsg = parseMsg("Unexpected log found", msg...)
	}
	if errMsg != "" {
		loggerFn("%s:%d %s: %v\nSearched logs:\n%v", filepath.Base(file), line, errMsg, want, logs)
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

func isDeepEqual(expected, actual interface{}) bool {
	_, isProto := expected.(proto.Message)
	if isProto {
		return proto.Equal(expected.(proto.Message), actual.(proto.Message))
	}
	return reflect.DeepEqual(expected, actual)
}

// NotEmpty asserts that an object's fields are not empty. This function recursively checks each
// pointer / struct field.
func NotEmpty(loggerFn assertionLoggerFn, obj interface{}, msg ...interface{}) {
	_, ignoreFieldsWithoutTags := obj.(proto.Message)
	notEmpty(loggerFn, obj, ignoreFieldsWithoutTags, []string{} /*fields*/, 0 /*stackSize*/, msg...)
}

// notEmpty checks all fields are not zero, including pointer field references to other structs.
// This method has the option to ignore fields without struct tags, which is helpful for checking
// protobuf messages that have internal fields.
func notEmpty(loggerFn assertionLoggerFn, obj interface{}, ignoreFieldsWithoutTags bool, fields []string, stackSize int, msg ...interface{}) {
	var v reflect.Value
	if vo, ok := obj.(reflect.Value); ok {
		v = reflect.Indirect(vo)
	} else {
		v = reflect.Indirect(reflect.ValueOf(obj))
	}

	if len(fields) == 0 {
		fields = []string{v.Type().Name()}
	}

	fail := func(fields []string) {
		m := parseMsg("", msg...)
		errMsg := fmt.Sprintf("empty/zero field: %s", strings.Join(fields, "."))
		if len(m) > 0 {
			m = strings.Join([]string{m, errMsg}, ": ")
		} else {
			m = errMsg
		}
		_, file, line, _ := runtime.Caller(4 + stackSize)
		loggerFn("%s:%d %s", filepath.Base(file), line, m)
	}

	if v.Kind() != reflect.Struct {
		if v.IsZero() {
			fail(fields)
		}
		return
	}

	for i := 0; i < v.NumField(); i++ {
		if ignoreFieldsWithoutTags && len(v.Type().Field(i).Tag) == 0 {
			continue
		}
		fields := append(fields, v.Type().Field(i).Name)

		switch k := v.Field(i).Kind(); k {
		case reflect.Ptr:
			notEmpty(loggerFn, v.Field(i), ignoreFieldsWithoutTags, fields, stackSize+1, msg...)
		case reflect.Slice:
			f := v.Field(i)
			if f.Len() == 0 {
				fail(fields)
			}
			for i := 0; i < f.Len(); i++ {
				notEmpty(loggerFn, f.Index(i), ignoreFieldsWithoutTags, fields, stackSize+1, msg...)
			}
		default:
			if v.Field(i).IsZero() {
				fail(fields)
			}
		}
	}
}

// TBMock exposes enough testing.TB methods for assertions.
type TBMock struct {
	ErrorfMsg string
	FatalfMsg string
}

// Errorf writes testing logs to ErrorfMsg.
func (tb *TBMock) Errorf(format string, args ...interface{}) {
	tb.ErrorfMsg = fmt.Sprintf(format, args...)
}

// Fatalf writes testing logs to FatalfMsg.
func (tb *TBMock) Fatalf(format string, args ...interface{}) {
	tb.FatalfMsg = fmt.Sprintf(format, args...)
}

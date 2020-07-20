// Package testutil defines common unit test utils such as asserting logs.
package testutil

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

// AssertLogsContain checks that the desired string is a subset of the current log output.
// Set exitOnFail to true to immediately exit the test on failure
func AssertLogsContain(t *testing.T, hook *test.Hook, want string) {
	assertLogs(t, hook, want, true)
}

// AssertLogsDoNotContain is the inverse check of AssertLogsContain
func AssertLogsDoNotContain(t *testing.T, hook *test.Hook, want string) {
	assertLogs(t, hook, want, false)
}

func assertLogs(t *testing.T, hook *test.Hook, want string, flag bool) {
	t.Logf("scanning for: %s", want)
	entries := hook.AllEntries()
	match := false
	for _, e := range entries {
		msg, err := e.String()
		if err != nil {
			t.Fatalf("Failed to format log entry to string: %v", err)
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
		t.Logf("log: %s", msg)
	}

	if flag && !match {
		t.Fatalf("log not found: %s", want)
	} else if !flag && match {
		t.Fatalf("unwanted log found: %s", want)
	}
}

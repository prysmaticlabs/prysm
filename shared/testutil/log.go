// Package testutil defines the testing utils such as asserting logs.
package testutil

import (
	"strings"
	"testing"
	"time"

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
		t.Logf("log: %s", msg)
	}

	if flag && !match {
		t.Fatalf("log not found: %s", want)
	} else if !flag && match {
		t.Fatalf("unwanted log found: %s", want)
	}
}

// WaitForLog waits for the desired string to appear the logs within a
// time period. If it does not appear within the limit, the function
// will throw an error.
func WaitForLog(t *testing.T, hook *test.Hook, want string) {
	t.Logf("waiting for: %s", want)
	match := false
	timer := time.After(1 * time.Second)

	for {
		select {
		case <-timer:
			t.Fatalf("log not found in time period: %s", want)
		default:
			if match {
				return
			}
			entries := hook.AllEntries()
			for _, e := range entries {
				if strings.Contains(e.Message, want) {
					match = true
				}
			}
		}
	}
}

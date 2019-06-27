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
	assertLogs(t, hook, want, true, -1)
}

// AssertLogsContainOccerances checks that the desired string is a subset of the current log output.
// Set exitOnFail to true to immediately exit the test on failure
func AssertLogsContainOccerances(t *testing.T, hook *test.Hook, want string, count int) {
	assertLogs(t, hook, want, true, count)
}

// AssertLogsDoNotContain is the inverse check of AssertLogsContain
func AssertLogsDoNotContain(t *testing.T, hook *test.Hook, want string) {
	assertLogs(t, hook, want, false, -1)
}

func assertLogs(t *testing.T, hook *test.Hook, want string, flag bool, count int) {
	t.Logf("scanning for: %s", want)
	entries := hook.AllEntries()
	match := false
	counter := 0
	for _, e := range entries {
		msg, err := e.String()
		if err != nil {
			t.Fatalf("Failed to format log entry to string: %v", err)
		}
		c := strings.Count(msg, want)
		if count != -1 {
			counter += c
		} else {
			if c > 0 {
				match = true
			}
		}

		t.Logf("log: %s", msg)
	}
	if count != -1 && counter == count {
		match = true
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

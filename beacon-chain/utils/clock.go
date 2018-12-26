package utils

import (
	"time"
)

// Clock represents a time providing interface that can be mocked for testing.
type Clock interface {
	Now() time.Time
}

// RealClock represents an unmodified clock.
type RealClock struct{}

// Now represents the standard functionality of time.
func (RealClock) Now() time.Time {
	return time.Now()
}

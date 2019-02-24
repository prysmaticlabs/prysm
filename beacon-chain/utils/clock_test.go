package utils

import (
	"testing"
	"time"
)

func TestRealClock_IsAccurate(t *testing.T) {
	var clock Clock = &RealClock{}
	clockTime := clock.Now().Second()
	actualTime := time.Now().Second()

	if clockTime != actualTime {
		t.Errorf("The time from the Clock interface should equal the actual time. Got: %v, Expected: %v", clockTime, actualTime)
	}
}

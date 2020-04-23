package logutil

import (
	"testing"
	"time"
)

func TestCountdownToGenesis(t *testing.T) {
	hook := logTest.NewGlobal()
	expectedStringResult := "01 minutes to genesis!\ngenesis time\n"
	result := countdownToGenesis(time.Now().Add(2*time.Second), 1)
	AssertLogsContain(t, hook, expectedStringResult)
	WaitForLog(t, hook, expectedStringResult, 2)
}

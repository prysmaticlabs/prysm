package logutil

import (
	"testing"
	"time"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestCountdownToGenesis(t *testing.T) {
	hook := logTest.NewGlobal()
	firstStringResult := "01 minutes to genesis!"
	CountdownToGenesis(roughtime.Now().Add(2*time.Second), 1)
	testutil.AssertLogsContain(t, hook, firstStringResult)
	testutil.WaitForLog(t, hook, firstStringResult, 1)
}

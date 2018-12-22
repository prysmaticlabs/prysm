package prometheus

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	prometheusService := NewPrometheusService(":2112")

	prometheusService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")

	prometheusService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

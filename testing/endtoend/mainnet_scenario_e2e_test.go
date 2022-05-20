package endtoend

import (
	"testing"
)

func TestEndToEnd_MainnetConfig_MultiClient(t *testing.T) {
	e2eMainnet(t, false /*usePrysmSh*/, true /*useMultiClient*/).run()
}

func TestEndToEnd_ScenarioRun_BeaconOffline_Multiclient(t *testing.T) {
	runner := e2eMainnet(t, false /*usePrysmSh*/, true /*useMultiClient*/)

	runner.config.Evaluators = scenarioEvals()
	runner.config.EvalInterceptor = runner.singleNodeOffline
	runner.scenarioRunner()
}

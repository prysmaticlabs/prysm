package endtoend

import (
	"testing"
)

func TestEndToEnd_MainnetConfig_MultiClient(t *testing.T) {
	e2eMainnet(t, false /*usePrysmSh*/, true /*useMultiClient*/).run()
}

func TestEndToEnd_ScenarioRun_BeaconOffline_Multiclient(t *testing.T) {
	runner := e2eMainnet(t, false /*usePrysmSh*/, true /*useMultiClient*/)
	runner.config.Evaluators = scenarioEvalsMulti()
	runner.config.EvalInterceptor = runner.singleNodeOffline
	runner.scenarioRunner()
}

func TestEndToEnd_ScenarioRun_OptimisticSync_Multiclient(t *testing.T) {
	runner := e2eMainnet(t, false /*usePrysmSh*/, true /*useMultiClient*/)
	runner.config.Evaluators = scenarioEvalsMulti()
	runner.config.EvalInterceptor = runner.optimisticSyncMulticlient
	runner.scenarioRunner()
}

package endtoend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
)

func TestEndToEnd_MainnetConfig_MultiClient(t *testing.T) {
	e2eMainnet(t, false, true, params.E2EMainnetTestConfig().Copy(), types.WithValidatorCrossClient()).run()
}

func TestEndToEnd_MultiScenarioRun_Multiclient(t *testing.T) {
	runner := e2eMainnet(t, false, true, params.E2EMainnetTestConfig().Copy(), types.WithEpochs(22))
	runner.config.Evaluators = scenarioEvalsMulti()
	runner.config.EvalInterceptor = runner.multiScenarioMulticlient
	runner.scenarioRunner()
}

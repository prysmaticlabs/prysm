package endtoend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
)

func TestEndToEnd_MultiScenarioRun_Multiclient(t *testing.T) {
	runner := e2eMainnet(t, false, true, types.StartAt(version.Phase0, params.E2EMainnetTestConfig()), types.WithEpochs(22))
	runner.config.Evaluators = scenarioEvalsMulti()
	runner.config.EvalInterceptor = runner.multiScenarioMulticlient
	runner.scenarioRunner()
}

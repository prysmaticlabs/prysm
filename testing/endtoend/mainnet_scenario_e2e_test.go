package endtoend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
)

func TestEndToEnd_MainnetConfig_MultiClient(t *testing.T) {
	e2eMainnet(t, false /*usePrysmSh*/, true /*useMultiClient*/).run()
}

func TestEndToEnd_MultiScenarioRun_Multiclient(t *testing.T) {
	t.Skip("Blocked until https://github.com/sigp/lighthouse/pull/3287 is merged in.")
	runner := e2eMainnet(t, false /*usePrysmSh*/, true /*useMultiClient*/, types.WithEpochs(22))
	runner.config.Evaluators = scenarioEvalsMulti()
	runner.config.EvalInterceptor = runner.multiScenarioMulticlient
	runner.scenarioRunner()
}

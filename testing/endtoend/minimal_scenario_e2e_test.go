package endtoend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
)

func TestEndToEnd_MultiScenarioRun(t *testing.T) {
	runner := e2eMinimal(t, types.WithEpochs(22))

	runner.config.Evaluators = scenarioEvals()
	runner.config.EvalInterceptor = runner.multiScenario
	runner.scenarioRunner()
}

func TestEndToEnd_MinimalConfig_Web3Signer(t *testing.T) {
	e2eMinimal(t, types.WithRemoteSigner()).run()
}

func TestEndToEnd_ScenarioRun_EEOffline(t *testing.T) {
	t.Skip("TODO(#10242) Prysm is current unable to handle an offline e2e")
	runner := e2eMinimal(t)

	runner.config.Evaluators = scenarioEvals()
	runner.config.EvalInterceptor = runner.eeOffline
	runner.scenarioRunner()
}

package endtoend

import "testing"

func TestEndToEnd_ScenarioRun_BeaconOffline(t *testing.T) {
	runner := e2eMinimal(t)

	runner.config.Evaluators = scenarioEvals()
	runner.config.EvalInterceptor = runner.singleNodeOffline
	runner.scenarioRunner()
}

func TestEndToEnd_ScenarioRun_AllvalidatorsOffline(t *testing.T) {
	runner := e2eMinimal(t)

	runner.config.Evaluators = scenarioEvals()
	runner.config.EvalInterceptor = runner.allValidatorsOffline
	runner.scenarioRunner()
}

func TestEndToEnd_ScenarioRun_EEOffline(t *testing.T) {
	t.Skip("TODO(#10242) Prysm is current unable to handle an offline e2e")
	runner := e2eMinimal(t)

	runner.config.Evaluators = scenarioEvals()
	runner.config.EvalInterceptor = runner.eeOffline
	runner.scenarioRunner()
}

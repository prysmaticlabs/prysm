package endtoend

import (
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/endtoend/components/eth1"
	ev "github.com/prysmaticlabs/prysm/testing/endtoend/evaluators"
	e2eParams "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestEndToEnd_Slasher_MinimalConfig(t *testing.T) {
	params.UseE2EConfig()
	require.NoError(t, e2eParams.Init(e2eParams.StandardBeaconCount))

	testConfig := &types.E2EConfig{
		BeaconFlags: []string{
			fmt.Sprintf("--slots-per-archive-point=%d", params.BeaconConfig().SlotsPerEpoch*16),
			fmt.Sprintf("--tracing-endpoint=http://%s", fmt.Sprintf("127.0.0.1:%d", 9411+e2eParams.TestParams.TestShardIndex)),
			"--enable-tracing",
			"--trace-sample-fraction=1.0",
		},
		ValidatorFlags:      []string{},
		EpochsToRun:         uint64(10),
		TestSync:            true,
		TestDeposits:        true,
		UsePrysmShValidator: false,
		UsePprof:            true,
		UseWeb3RemoteSigner: false,
		TracingSinkEndpoint: fmt.Sprintf("127.0.0.1:%d", 9411+e2eParams.TestParams.TestShardIndex),
		Evaluators: []types.Evaluator{
			ev.PeersConnect,
			ev.HealthzCheck,
			ev.ValidatorsSlashedAfterEpoch(4),
			ev.SlashedValidatorsLoseBalanceAfterEpoch(4),
			ev.InjectDoubleVoteOnEpoch(2),
			ev.InjectDoubleBlockOnEpoch(2),
		},
		Port: eth1.MinerPort + 1,
	}

	newTestRunner(t, testConfig).run()
}

package endtoend

import (
	"context"
	"os"
	"strconv"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	slashersimulator "github.com/prysmaticlabs/prysm/testing/slasher/simulator"
	"github.com/prysmaticlabs/prysm/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockSyncChecker struct{}

func (c mockSyncChecker) Initialized() bool {
	return true
}

func (c mockSyncChecker) Syncing() bool {
	return false
}

func (c mockSyncChecker) Synced() bool {
	return true
}

func (c mockSyncChecker) Status() error {
	return nil
}

func (c mockSyncChecker) Resync() error {
	return nil
}

func (mockSyncChecker) IsSynced(_ context.Context) (bool, error) {
	return true, nil
}

func TestEndToEnd_SlasherSimulator(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()

	// Run for 10 epochs if not in long-running to confirm long-running has no issues.
	simulatorParams := slashersimulator.DefaultParams()
	var err error
	epochStr, longRunning := os.LookupEnv("E2E_EPOCHS")
	if longRunning {
		epochsToRun, err := strconv.Atoi(epochStr)
		require.NoError(t, err)
		simulatorParams.NumEpochs = uint64(epochsToRun)
	}

	slasherDB := dbtest.SetupSlasherDB(t)
	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)

	// We setup validators in the beacon state along with their
	// private keys used to generate valid signatures in generated objects.
	validators := make([]*ethpb.Validator, simulatorParams.NumValidators)
	privKeys := make(map[types.ValidatorIndex]bls.SecretKey)
	for valIdx := range validators {
		privKey, err := bls.RandKey()
		require.NoError(t, err)
		privKeys[types.ValidatorIndex(valIdx)] = privKey
		validators[valIdx] = &ethpb.Validator{
			PublicKey:             privKey.PublicKey().Marshal(),
			WithdrawalCredentials: make([]byte, 32),
		}
	}
	err = beaconState.SetValidators(validators)
	require.NoError(t, err)

	mockChain := &mock.ChainService{State: beaconState}
	gen := stategen.NewMockService()
	gen.AddStateForRoot(beaconState, [32]byte{})

	sim, err := slashersimulator.New(ctx, &slashersimulator.ServiceConfig{
		Params:                      simulatorParams,
		Database:                    slasherDB,
		StateNotifier:               &mock.MockStateNotifier{},
		HeadStateFetcher:            mockChain,
		AttestationStateFetcher:     mockChain,
		StateGen:                    gen,
		PrivateKeysByValidatorIndex: privKeys,
		SlashingsPool:               &slashings.PoolMock{},
		SyncChecker:                 mockSyncChecker{},
	})
	require.NoError(t, err)
	sim.Start()
	err = sim.Stop()
	require.NoError(t, err)
	require.LogsDoNotContain(t, hook, "ERROR")
	require.LogsDoNotContain(t, hook, "Did not detect")
	require.LogsContain(t, hook, "Correctly detected simulated proposer slashing")
	require.LogsContain(t, hook, "Correctly detected simulated attester slashing")
}

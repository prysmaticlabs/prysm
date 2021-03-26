package p2p

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestCorrect_ActiveValidatorsCount(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.MainnetConfig()
	cfg.ConfigName = "test"

	params.OverrideBeaconConfig(cfg)

	db := dbutil.SetupDB(t)
	s := &Service{
		ctx: context.Background(),
		cfg: &Config{DB: db},
	}
	bState, err := testutil.NewBeaconState(func(state *pb.BeaconState) error {
		validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
		for i := 0; i < len(validators); i++ {
			validators[i] = &ethpb.Validator{
				PublicKey:             make([]byte, 48),
				WithdrawalCredentials: make([]byte, 32),
				ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
				Slashed:               false,
			}
		}
		state.Validators = validators
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisData(s.ctx, bState))

	vals, err := s.retrieveActiveValidators()
	assert.NoError(t, err, "genesis state not retrieved")
	assert.Equal(t, int(params.BeaconConfig().MinGenesisActiveValidatorCount), int(vals), "mainnet genesis active count isn't accurate")
	for i := 0; i < 100; i++ {
		require.NoError(t, bState.AppendValidator(&ethpb.Validator{
			PublicKey:             make([]byte, 48),
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			Slashed:               false,
		}))
	}
	require.NoError(t, bState.SetSlot(10000))
	require.NoError(t, db.SaveState(s.ctx, bState, [32]byte{'a'}))

	// Retrieve last archived state.
	vals, err = s.retrieveActiveValidators()
	assert.NoError(t, err, "genesis state not retrieved")
	assert.Equal(t, int(params.BeaconConfig().MinGenesisActiveValidatorCount)+100, int(vals), "mainnet genesis active count isn't accurate")
}

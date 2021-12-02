package util

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v3 "github.com/prysmaticlabs/prysm/beacon-chain/state/v3"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// DeterministicGenesisStateMerge returns a genesis state in Merge format made using the deterministic deposits.
func DeterministicGenesisStateMerge(t testing.TB, numValidators uint64) (state.BeaconState, []bls.SecretKey) {
	deposits, privKeys, err := DeterministicDepositsAndKeys(numValidators)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get %d deposits", numValidators))
	}
	eth1Data, err := DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get eth1data for %d deposits", numValidators))
	}
	beaconState, err := genesisBeaconStateMerge(context.Background(), deposits, uint64(0), eth1Data)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get genesis beacon state of %d validators", numValidators))
	}
	resetCache()
	return beaconState, privKeys
}

// genesisBeaconStateMerge returns the genesis beacon state.
func genesisBeaconStateMerge(ctx context.Context, deposits []*ethpb.Deposit, genesisTime uint64, eth1Data *ethpb.Eth1Data) (state.BeaconState, error) {
	st, err := emptyGenesisStateMerge()
	if err != nil {
		return nil, err
	}

	// Process initial deposits.
	st, err = helpers.UpdateGenesisEth1Data(st, deposits, eth1Data)
	if err != nil {
		return nil, err
	}

	st, err = processPreGenesisDeposits(ctx, st, deposits)
	if err != nil {
		return nil, errors.Wrap(err, "could not process validator deposits")
	}

	return buildGenesisBeaconState(genesisTime, st, st.Eth1Data())
}

// emptyGenesisStateMerge returns an empty genesis state in Merge format.
func emptyGenesisStateMerge() (state.BeaconState, error) {
	st := &ethpb.BeaconStateMerge{
		// Misc fields.
		Slot: 0,
		Fork: &ethpb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().AltairForkVersion,
			Epoch:           0,
		},
		// Validator registry fields.
		Validators:       []*ethpb.Validator{},
		Balances:         []uint64{},
		InactivityScores: []uint64{},

		JustificationBits:          []byte{0},
		HistoricalRoots:            [][]byte{},
		CurrentEpochParticipation:  []byte{},
		PreviousEpochParticipation: []byte{},

		// Eth1 data.
		Eth1Data:         &ethpb.Eth1Data{},
		Eth1DataVotes:    []*ethpb.Eth1Data{},
		Eth1DepositIndex: 0,

		LatestExecutionPayloadHeader: &ethpb.ExecutionPayloadHeader{},
	}
	return v3.InitializeFromProto(st)
}

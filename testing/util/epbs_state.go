package util

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// emptyGenesisStateEpbs returns an empty genesis state in ePBS format.
func emptyGenesisStateEpbs() (state.BeaconState, error) {
	st := &ethpb.BeaconStateEPBS{
		// Misc fields.
		Slot: 0,
		Fork: &ethpb.Fork{
			PreviousVersion: params.BeaconConfig().BellatrixForkVersion,
			CurrentVersion:  params.BeaconConfig().DenebForkVersion,
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

		LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeaderEPBS{},

		DepositBalanceToConsume:       primitives.Gwei(0),
		ExitBalanceToConsume:          primitives.Gwei(0),
		ConsolidationBalanceToConsume: primitives.Gwei(0),
	}
	return state_native.InitializeFromProtoEpbs(st)
}

// genesisBeaconStateEpbs returns the genesis beacon state.
func genesisBeaconStateEpbs(ctx context.Context, deposits []*ethpb.Deposit, genesisTime uint64, eth1Data *ethpb.Eth1Data) (state.BeaconState, error) {
	st, err := emptyGenesisStateElectra()
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

	return buildGenesisBeaconStateElectra(genesisTime, st, st.Eth1Data())
}

// DeterministicGenesisStateEpbs returns a genesis state in ePBS format made using the deterministic deposits.
func DeterministicGenesisStateEpbs(t testing.TB, numValidators uint64) (state.BeaconState, []bls.SecretKey) {
	deposits, privKeys, err := DeterministicDepositsAndKeys(numValidators)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get %d deposits", numValidators))
	}
	eth1Data, err := DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get eth1data for %d deposits", numValidators))
	}
	beaconState, err := genesisBeaconStateElectra(context.Background(), deposits, uint64(0), eth1Data)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get genesis beacon state of %d validators", numValidators))
	}
	if err := setKeysToActive(beaconState); err != nil {
		t.Fatal(errors.Wrapf(err, "failed to set keys to active"))
	}
	resetCache()
	return beaconState, privKeys
}

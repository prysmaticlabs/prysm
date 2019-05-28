// Package state implements the whole state transition
// function which consists of per slot, per-epoch transitions.
// It also bootstraps the genesis beacon state for slot 0.
package state

import (
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// GenesisBeaconState gets called when DepositsForChainStart count of
// full deposits were made to the deposit contract and the ChainStart log gets emitted.
func GenesisBeaconState(
	genesisValidatorDeposits []*pb.Deposit,
	genesisTime uint64,
	eth1Data *pb.Eth1Data,
) (*pb.BeaconState, error) {
	latestRandaoMixes := make(
		[][]byte,
		params.BeaconConfig().LatestRandaoMixesLength,
	)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = make([]byte, 32)
	}

	zeroHash := params.BeaconConfig().ZeroHash[:]

	latestActiveIndexRoots := make(
		[][]byte,
		params.BeaconConfig().LatestActiveIndexRootsLength,
	)
	for i := 0; i < len(latestActiveIndexRoots); i++ {
		latestActiveIndexRoots[i] = zeroHash
	}

	crosslinks := make([]*pb.Crosslink, params.BeaconConfig().ShardCount)
	for i := 0; i < len(crosslinks); i++ {
		crosslinks[i] = &pb.Crosslink{
			Shard: uint64(i),
		}
	}

	latestBlockRoots := make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	for i := 0; i < len(latestBlockRoots); i++ {
		latestBlockRoots[i] = zeroHash
	}

	validatorRegistry := make([]*pb.Validator, len(genesisValidatorDeposits))
	for i, d := range genesisValidatorDeposits {

		validator := &pb.Validator{
			Pubkey:                d.Data.Pubkey,
			WithdrawalCredentials: d.Data.WithdrawalCredentials,
			ActivationEpoch:       params.BeaconConfig().FarFutureEpoch,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			Slashed:               false,
			WithdrawableEpoch:     params.BeaconConfig().FarFutureEpoch,
		}

		validatorRegistry[i] = validator
	}

	latestBalances := make([]uint64, len(genesisValidatorDeposits))
	latestSlashedExitBalances := make([]uint64, params.BeaconConfig().LatestSlashedExitLength)

	state := &pb.BeaconState{
		// Misc fields.
		Slot:        params.BeaconConfig().GenesisSlot,
		GenesisTime: genesisTime,

		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			Epoch:           params.BeaconConfig().GenesisEpoch,
		},

		// Validator registry fields.
		ValidatorRegistry: validatorRegistry,
		Balances:          latestBalances,

		// Randomness and committees.
		LatestRandaoMixes: latestRandaoMixes,

		// Finality.
		PreviousJustifiedEpoch: params.BeaconConfig().GenesisEpoch,
		PreviousJustifiedRoot:  params.BeaconConfig().ZeroHash[:],
		CurrentJustifiedEpoch:  params.BeaconConfig().GenesisEpoch,
		CurrentJustifiedRoot:   params.BeaconConfig().ZeroHash[:],
		JustificationBitfield:  0,
		FinalizedEpoch:         params.BeaconConfig().GenesisEpoch,
		FinalizedRoot:          params.BeaconConfig().ZeroHash[:],

		// Recent state.
		CurrentCrosslinks:         crosslinks,
		PreviousCrosslinks:        crosslinks,
		LatestActiveIndexRoots:    latestActiveIndexRoots,
		LatestBlockRoots:          latestBlockRoots,
		LatestSlashedBalances:     latestSlashedExitBalances,
		CurrentEpochAttestations:  []*pb.PendingAttestation{},
		PreviousEpochAttestations: []*pb.PendingAttestation{},

		// Eth1 data.
		LatestEth1Data: eth1Data,
		Eth1DataVotes:  []*pb.Eth1Data{},
		DepositIndex:   0,
	}

	// Process initial deposits.
	var err error
	validatorMap := stateutils.ValidatorIndexMap(state)
	for _, deposit := range genesisValidatorDeposits {
		state, err = v.ProcessDeposit(
			state,
			validatorMap,
			deposit.Data.Pubkey,
			deposit.Data.Amount,
			deposit.Data.Signature,
			deposit.Data.WithdrawalCredentials,
		)
		if err != nil {
			return nil, fmt.Errorf("could not process validator deposit: %v", err)
		}
	}
	for i := 0; i < len(state.ValidatorRegistry); i++ {
		if state.ValidatorRegistry[i].EffectiveBalance >=
			params.BeaconConfig().MaxDepositAmount {
			state, err = v.ActivateValidator(state, uint64(i), true)
			if err != nil {
				return nil, fmt.Errorf("could not activate validator: %v", err)
			}
		}
	}
	activeValidators := helpers.ActiveValidatorIndices(state, params.BeaconConfig().GenesisEpoch)
	indicesBytes := []byte{}
	for _, val := range activeValidators {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, val)
		indicesBytes = append(indicesBytes, buf...)
	}
	genesisActiveIndexRoot := hashutil.Hash(indicesBytes)
	for i := uint64(0); i < params.BeaconConfig().LatestActiveIndexRootsLength; i++ {
		state.LatestActiveIndexRoots[i] = genesisActiveIndexRoot[:]
	}
	seed, err := helpers.GenerateSeed(state, params.BeaconConfig().GenesisEpoch)
	if err != nil {
		return nil, fmt.Errorf("could not generate initial seed: %v", err)
	}
	state.CurrentShufflingSeedHash32 = seed[:]
	return state, nil
}

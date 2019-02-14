// Package state implements the whole state transition
// function which consists of per slot, per-epoch transitions.
// It also bootstraps the genesis beacon state for slot 0.
package state

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/ssz"

	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// InitialBeaconState gets called when DepositsForChainStart count of
// full deposits were made to the deposit contract and the ChainStart log gets emitted.
func InitialBeaconState(
	initialValidatorDeposits []*pb.Deposit,
	genesisTime uint64,
	processedPowReceiptRoot []byte,
) (*pb.BeaconState, error) {
	latestRandaoMixes := make(
		[][]byte,
		params.BeaconConfig().LatestRandaoMixesLength,
	)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}

	latestIndexRoots := make(
		[][]byte,
		params.BeaconConfig().LatestIndexRootsLength,
	)
	for i := 0; i < len(latestIndexRoots); i++ {
		latestIndexRoots[i] = params.BeaconConfig().ZeroHash[:]
	}

	latestVDFOutputs := make([][]byte,
		params.BeaconConfig().LatestRandaoMixesLength/params.BeaconConfig().EpochLength)
	for i := 0; i < len(latestVDFOutputs); i++ {
		latestVDFOutputs[i] = params.BeaconConfig().ZeroHash[:]
	}

	latestCrosslinks := make([]*pb.Crosslink, params.BeaconConfig().ShardCount)
	for i := 0; i < len(latestCrosslinks); i++ {
		latestCrosslinks[i] = &pb.Crosslink{
			Epoch:                params.BeaconConfig().GenesisEpoch,
			ShardBlockRootHash32: params.BeaconConfig().ZeroHash[:],
		}
	}

	latestBlockRoots := make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	for i := 0; i < len(latestBlockRoots); i++ {
		latestBlockRoots[i] = params.BeaconConfig().ZeroHash[:]
	}

	validatorRegistry := make([]*pb.Validator, len(initialValidatorDeposits))
	latestBalances := make([]uint64, len(initialValidatorDeposits))
	for i, d := range initialValidatorDeposits {
		depositInput, err := b.DecodeDepositInput(d.DepositData)
		if err != nil {
			return nil, fmt.Errorf("could decode deposit input %v", err)
		}

		validator := &pb.Validator{
			Pubkey:                      depositInput.Pubkey,
			WithdrawalCredentialsHash32: depositInput.WithdrawalCredentialsHash32,
			ExitEpoch:                   params.BeaconConfig().FarFutureEpoch,
			PenalizedEpoch:              params.BeaconConfig().FarFutureEpoch,
		}

		validatorRegistry[i] = validator

	}

	latestPenalizedExitBalances := make([]uint64, params.BeaconConfig().LatestPenalizedExitLength)

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
		ValidatorRegistry:            validatorRegistry,
		ValidatorBalances:            latestBalances,
		ValidatorRegistryUpdateEpoch: params.BeaconConfig().GenesisEpoch,

		// Randomness and committees.
		LatestRandaoMixesHash32S: latestRandaoMixes,
		PreviousEpochStartShard:  params.BeaconConfig().GenesisStartShard,
		CurrentEpochStartShard:   params.BeaconConfig().GenesisStartShard,
		PreviousCalculationEpoch: params.BeaconConfig().GenesisEpoch,
		CurrentCalculationEpoch:  params.BeaconConfig().GenesisEpoch,
		PreviousEpochSeedHash32:  params.BeaconConfig().ZeroHash[:],
		CurrentEpochSeedHash32:   params.BeaconConfig().ZeroHash[:],

		// Finality.
		PreviousJustifiedEpoch: params.BeaconConfig().GenesisEpoch,
		JustifiedEpoch:         params.BeaconConfig().GenesisEpoch,
		JustificationBitfield:  0,
		FinalizedEpoch:         params.BeaconConfig().GenesisEpoch,

		// Recent state.
		LatestCrosslinks:        latestCrosslinks,
		LatestBlockRootHash32S:  latestBlockRoots,
		LatestIndexRootHash32S:  latestIndexRoots,
		LatestPenalizedBalances: latestPenalizedExitBalances,
		LatestAttestations:      []*pb.PendingAttestation{},
		BatchedBlockRootHash32S: [][]byte{},

		// Eth1 data.
		LatestEth1Data: &pb.Eth1Data{
			DepositRootHash32: processedPowReceiptRoot,
			BlockHash32:       []byte{},
		},
		Eth1DataVotes: []*pb.Eth1DataVote{},
	}

	// Process initial deposits.
	var err error
	validatorMap := stateutils.ValidatorIndexMap(state)
	for _, deposit := range initialValidatorDeposits {
		depositData := deposit.DepositData
		depositInput, err := b.DecodeDepositInput(depositData)
		if err != nil {
			return nil, fmt.Errorf("could not decode deposit input: %v", err)
		}
		value, _, err := b.DecodeDepositAmountAndTimeStamp(depositData)
		if err != nil {
			return nil, fmt.Errorf("could not decode deposit value and timestamp: %v", err)
		}
		state, err = v.ProcessDeposit(
			state,
			validatorMap,
			depositInput.Pubkey,
			value,
			depositInput.ProofOfPossession,
			depositInput.WithdrawalCredentialsHash32,
		)
		if err != nil {
			return nil, fmt.Errorf("could not process validator deposit: %v", err)
		}
	}
	for i := 0; i < len(state.ValidatorRegistry); i++ {
		if helpers.EffectiveBalance(state, uint64(i)) >=
			params.BeaconConfig().MaxDepositAmount {
			state, err = v.ActivateValidator(state, uint64(i), true)
			if err != nil {
				return nil, fmt.Errorf("could not activate validator: %v", err)
			}
		}
	}
	activeValidators := helpers.ActiveValidatorIndices(state.ValidatorRegistry, params.BeaconConfig().GenesisEpoch)
	genesisActiveIndexRoot, err := ssz.TreeHash(activeValidators)
	if err != nil {
		return nil, fmt.Errorf("could not determine genesis active index root: %v", err)
	}
	for i := uint64(0); i < params.BeaconConfig().LatestIndexRootsLength; i++ {
		state.LatestIndexRootHash32S[i] = genesisActiveIndexRoot[:]
	}
	seed, err := helpers.GenerateSeed(state, params.BeaconConfig().GenesisEpoch)
	if err != nil {
		return nil, fmt.Errorf("could not generate initial seed: %v", err)
	}
	state.CurrentEpochSeedHash32 = seed[:]
	return state, nil
}

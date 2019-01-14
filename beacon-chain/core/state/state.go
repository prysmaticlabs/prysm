// Package state implements the whole state transition
// function which consists of per slot, per-epoch transitions.
// It also bootstraps the genesis beacon state for slot 0.
package state

import (
	"encoding/binary"
	"fmt"

	"github.com/gogo/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var config = params.BeaconConfig()

// InitialBeaconState gets called when DepositsForChainStart count of
// full deposits were made to the deposit contract and the ChainStart log gets emitted.
func InitialBeaconState(
	initialValidatorDeposits []*pb.Deposit,
	genesisTime uint64,
	processedPowReceiptRoot []byte,
) (*pb.BeaconState, error) {
	latestRandaoMixes := make(
		[][]byte,
		config.LatestRandaoMixesLength,
	)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = config.ZeroHash[:]
	}

	latestVDFOutputs := make([][]byte,
		config.LatestRandaoMixesLength/config.EpochLength)
	for i := 0; i < len(latestVDFOutputs); i++ {
		latestVDFOutputs[i] = config.ZeroHash[:]
	}

	latestCrosslinks := make([]*pb.CrosslinkRecord, config.ShardCount)
	for i := 0; i < len(latestCrosslinks); i++ {
		latestCrosslinks[i] = &pb.CrosslinkRecord{
			Slot:                 config.GenesisSlot,
			ShardBlockRootHash32: config.ZeroHash[:],
		}
	}

	latestBlockRoots := make([][]byte, config.LatestBlockRootsLength)
	for i := 0; i < len(latestBlockRoots); i++ {
		latestBlockRoots[i] = config.ZeroHash[:]
	}

	validatorRegistry := make([]*pb.ValidatorRecord, len(initialValidatorDeposits))
	latestBalances := make([]uint64, len(initialValidatorDeposits))
	for i, d := range initialValidatorDeposits {

		amount, _, err := b.DecodeDepositAmountAndTimeStamp(d.DepositData)
		if err != nil {
			return nil, fmt.Errorf("could not decode deposit amount and timestamp %v", err)
		}

		depositInput, err := b.DecodeDepositInput(d.DepositData)
		if err != nil {
			return nil, fmt.Errorf("could decode deposit input %v", err)
		}

		validator := &pb.ValidatorRecord{
			Pubkey:                      depositInput.Pubkey,
			RandaoCommitmentHash32:      depositInput.RandaoCommitmentHash32,
			WithdrawalCredentialsHash32: depositInput.WithdrawalCredentialsHash32,
			CustodyCommitmentHash32:     depositInput.CustodyCommitmentHash32,
			Balance:                     amount,
			ExitSlot:                    config.FarFutureSlot,
		}

		validatorRegistry[i] = validator

	}

	latestPenalizedExitBalances := make([]uint64, config.LatestPenalizedExitLength)

	state := &pb.BeaconState{
		// Misc fields.
		Slot:        config.GenesisSlot,
		GenesisTime: genesisTime,
		ForkData: &pb.ForkData{
			PreForkVersion:  config.GenesisForkVersion,
			PostForkVersion: config.GenesisForkVersion,
			ForkSlot:        config.GenesisSlot,
		},

		// Validator registry fields.
		ValidatorRegistry:                    validatorRegistry,
		ValidatorBalances:                    latestBalances,
		ValidatorRegistryLatestChangeSlot:    config.GenesisSlot,
		ValidatorRegistryExitCount:           0,
		ValidatorRegistryDeltaChainTipHash32: config.ZeroHash[:],

		// Randomness and committees.
		LatestRandaoMixesHash32S:     latestRandaoMixes,
		LatestVdfOutputsHash32S:      latestVDFOutputs,
		PreviousEpochStartShard:      config.GenesisStartShard,
		CurrentEpochStartShard:       config.GenesisStartShard,
		PreviousEpochCalculationSlot: config.GenesisSlot,
		CurrentEpochCalculationSlot:  config.GenesisSlot,
		PreviousEpochRandaoMixHash32: config.ZeroHash[:],
		CurrentEpochRandaoMixHash32:  config.ZeroHash[:],

		// Proof of custody.
		// Place holder, proof of custody challenge is defined in phase 1.
		// This list will remain empty through out phase 0.
		CustodyChallenges: []*pb.CustodyChallenge{},

		// Finality.
		PreviousJustifiedSlot: config.GenesisSlot,
		JustifiedSlot:         config.GenesisSlot,
		JustificationBitfield: 0,
		FinalizedSlot:         config.GenesisSlot,

		// Recent state.
		LatestCrosslinks:            latestCrosslinks,
		LatestBlockRootHash32S:      latestBlockRoots,
		LatestPenalizedExitBalances: latestPenalizedExitBalances,
		LatestAttestations:          []*pb.PendingAttestationRecord{},
		BatchedBlockRootHash32S:     [][]byte{},

		// deposit root.
		LatestDepositRootHash32: processedPowReceiptRoot,
		DepositRootVotes:        []*pb.DepositRootVote{},
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
		// depositData consists of depositInput []byte + depositValue [8]byte +
		// depositTimestamp [8]byte.
		depositValue := depositData[len(depositData)-16 : len(depositData)-8]
		state, err = v.ProcessDeposit(
			state,
			validatorMap,
			depositInput.Pubkey,
			binary.BigEndian.Uint64(depositValue),
			depositInput.ProofOfPossession,
			depositInput.WithdrawalCredentialsHash32,
			depositInput.RandaoCommitmentHash32,
			depositInput.CustodyCommitmentHash32,
		)
		if err != nil {
			return nil, fmt.Errorf("could not process validator deposit: %v", err)
		}
	}
	for i := 0; i < len(state.ValidatorRegistry); i++ {
		if v.EffectiveBalance(state, uint32(i)) ==
			config.MaxDepositInGwei {
			state, err = v.ActivateValidator(state, uint32(i), true)
			if err != nil {
				return nil, fmt.Errorf("could not activate validator: %v", err)
			}
		}
	}

	return state, nil
}

// Hash the beacon state data structure.
func Hash(state *pb.BeaconState) ([32]byte, error) {
	data, err := proto.Marshal(state)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not marshal beacon state: %v", err)
	}
	return hashutil.Hash(data), nil
}

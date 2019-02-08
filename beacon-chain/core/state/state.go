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

	latestCrosslinks := make([]*pb.CrosslinkRecord, params.BeaconConfig().ShardCount)
	for i := 0; i < len(latestCrosslinks); i++ {
		latestCrosslinks[i] = &pb.CrosslinkRecord{
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
			RandaoCommitmentHash32:      depositInput.RandaoCommitmentHash32,
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
		LatestIndexRootHash32S:   latestIndexRoots,
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
		LatestPenalizedBalances: latestPenalizedExitBalances,
		LatestAttestations:      []*pb.PendingAttestationRecord{},
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
		)
		if err != nil {
			return nil, fmt.Errorf("could not process validator deposit: %v", err)
		}
	}
	for i := 0; i < len(state.ValidatorRegistry); i++ {
		if v.EffectiveBalance(state, uint64(i)) >=
			params.BeaconConfig().MaxDeposit {
			state, err = v.ActivateValidator(state, uint64(i), true)
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

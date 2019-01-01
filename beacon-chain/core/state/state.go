package state

import (
	"encoding/binary"
	"fmt"

	"github.com/gogo/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pbcomm "github.com/prysmaticlabs/prysm/proto/common"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// InitialBeaconState gets called when DepositsForChainStart count of
// full deposits were made to the deposit contract and the ChainStart log gets emitted.
func InitialBeaconState(
	initialValidatorDeposits []*pb.Deposit,
	genesisTime uint64,
	processedPowReceiptRoot []byte) (*pb.BeaconState, error) {
	latestRandaoMixes := make([][]byte,
		params.BeaconConfig().LatestRandaoMixesLength)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}

	latestVDFOutputs := make([][]byte,
		params.BeaconConfig().LatestRandaoMixesLength/params.BeaconConfig().EpochLength)
	for i := 0; i < len(latestVDFOutputs); i++ {
		latestVDFOutputs[i] = params.BeaconConfig().ZeroHash[:]
	}

	latestCrosslinks := make([]*pb.CrosslinkRecord, params.BeaconConfig().ShardCount)
	for i := 0; i < len(latestCrosslinks); i++ {
		latestCrosslinks[i] = &pb.CrosslinkRecord{
			Slot:                 params.BeaconConfig().InitialSlotNumber,
			ShardBlockRootHash32: params.BeaconConfig().ZeroHash[:],
		}
	}

	latestBlockRoots := make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	for i := 0; i < len(latestBlockRoots); i++ {
		latestBlockRoots[i] = params.BeaconConfig().ZeroHash[:]
	}

	state := &pb.BeaconState{
		// Misc fields.
		Slot:        params.BeaconConfig().InitialSlotNumber,
		GenesisTime: genesisTime,
		ForkData: &pb.ForkData{
			PreForkVersion:  params.BeaconConfig().InitialForkVersion,
			PostForkVersion: params.BeaconConfig().InitialForkVersion,
			ForkSlot:        params.BeaconConfig().InitialSlotNumber,
		},

		// Validator registry fields.
		ValidatorRegistry:                    []*pb.ValidatorRecord{},
		ValidatorBalances:                    []uint64{},
		ValidatorRegistryLastChangeSlot:      params.BeaconConfig().InitialSlotNumber,
		ValidatorRegistryExitCount:           0,
		ValidatorRegistryDeltaChainTipHash32: params.BeaconConfig().ZeroHash[:],

		// Randomness and committees.
		LatestRandaoMixesHash32S:         latestRandaoMixes,
		LatestVdfOutputs:                 latestVDFOutputs,
		ShardAndCommitteesAtSlots:        []*pb.ShardAndCommitteeArray{},
		PersistentCommittees:             []*pbcomm.Uint32List{},
		PersistentCommitteeReassignments: []*pb.ShardReassignmentRecord{},

		// Proof of custody.
		// Place holder, proof of custody challenge is defined in phase 1.
		// This list will remain empty through out phase 0.
		PocChallenges: []*pb.ProofOfCustodyChallenge{},

		// Finality.
		PreviousJustifiedSlot: params.BeaconConfig().InitialSlotNumber,
		JustifiedSlot:         params.BeaconConfig().InitialSlotNumber,
		JustificationBitfield: 0,
		FinalizedSlot:         params.BeaconConfig().InitialSlotNumber,

		// Recent state.
		LatestCrosslinks:            latestCrosslinks,
		LatestBlockRootHash32S:      latestBlockRoots,
		LatestPenalizedExitBalances: []uint64{},
		LatestAttestations:          []*pb.PendingAttestationRecord{},
		BatchedBlockRootHash32S:     [][]byte{},

		// PoW receipt root.
		ProcessedPowReceiptRootHash32: processedPowReceiptRoot,
		CandidatePowReceiptRoots:      []*pb.CandidatePoWReceiptRootRecord{},
	}

	// Set initial deposits and activations.
	var validatorIndex uint32
	var err error
	for _, deposit := range initialValidatorDeposits {
		depositData := deposit.GetDepositData()
		depositInput, err := b.DecodeDepositInput(depositData)
		if err != nil {
			return nil, fmt.Errorf("could not decode deposit input: %v", err)
		}
		// depositData consists of depositInput []byte + depositValue [8]byte +
		// depositTimestamp [8]byte.
		depositValue := depositData[len(depositData)-16 : len(depositData)-8]
		state, validatorIndex, err = v.ProcessDeposit(
			state,
			depositInput.GetPubkey(),
			binary.BigEndian.Uint64(depositValue),
			depositInput.GetProofOfPossession(),
			depositInput.GetWithdrawalCredentialsHash32(),
			depositInput.GetRandaoCommitmentHash32(),
			depositInput.GetPocCommitment(),
		)
		if err != nil {
			return nil, fmt.Errorf("could not process validator deposit: %v", err)
		}
		if v.EffectiveBalance(state, validatorIndex) ==
			params.BeaconConfig().MaxDepositInGwei {
			state, err = v.UpdateStatus(state, validatorIndex, pb.ValidatorRecord_ACTIVE)
			if err != nil {
				return nil, fmt.Errorf("could not update validator status: %v", err)
			}
		}
	}

	// Set initial committee shuffling.
	initialShuffling, err := v.ShuffleValidatorRegistryToCommittees(
		params.BeaconConfig().ZeroHash,
		state.ValidatorRegistry,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("could not shuffle initial committee: %v", err)
	}
	state.ShardAndCommitteesAtSlots = append(initialShuffling, initialShuffling...)

	// Set initial persistent shuffling.
	activeValidatorIndices := v.ActiveValidatorIndices(state.ValidatorRegistry)
	committees := utils.SplitIndices(activeValidatorIndices, params.BeaconConfig().ShardCount)
	persistentCommittees := make([]*pbcomm.Uint32List, params.BeaconConfig().ShardCount)
	for i := 0; i < len(persistentCommittees); i++ {
		persistentCommittees[i] = &pbcomm.Uint32List{List: committees[i]}
	}
	state.PersistentCommittees = persistentCommittees

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

// CalculateNewBlockHashes builds a new slice of recent block hashes with the
// provided block and the parent slot number.
//
// The algorithm is:
//   1) shift the array by block.SlotNumber - parentSlot (i.e. truncate the
//     first by the number of slots that have occurred between the block and
//     its parent).
//
//   2) fill the array with the parent block hash for all values between the parent
//     slot and the block slot.
//
// Computation of the state hash depends on this feature that slots with
// missing blocks have the block hash of the next block hash in the chain.
//
// For example, if we have a segment of recent block hashes that look like this
//   [0xF, 0x7, 0x0, 0x0, 0x5]
//
// Where 0x0 is an empty or missing hash where no block was produced in the
// alloted slot. When storing the list (or at least when computing the hash of
// the active state), the list should be back-filled as such:
//
//   [0xF, 0x7, 0x5, 0x5, 0x5]
func CalculateNewBlockHashes(state *pb.BeaconState, block *pb.BeaconBlock, parentSlot uint64) ([][]byte, error) {
	distance := block.GetSlot() - parentSlot
	existing := state.GetLatestBlockRootHash32S()
	update := existing[distance:]
	for len(update) < 2*int(params.BeaconConfig().CycleLength) {
		update = append(update, block.GetParentRootHash32())
	}
	return update, nil
}

// IsValidatorSetChange checks if a validator set change transition can be processed. At that point,
// validator shuffle will occur.
func IsValidatorSetChange(state *pb.BeaconState, slotNumber uint64) bool {
	if state.GetFinalizedSlot() <= state.GetValidatorRegistryLastChangeSlot() {
		return false
	}
	if slotNumber-state.GetValidatorRegistryLastChangeSlot() < params.BeaconConfig().MinValidatorSetChangeInterval {
		return false
	}

	shardProcessed := map[uint64]bool{}
	for _, shardAndCommittee := range state.GetShardAndCommitteesAtSlots() {
		for _, committee := range shardAndCommittee.ArrayShardAndCommittee {
			shardProcessed[committee.Shard] = true
		}
	}

	crosslinks := state.GetLatestCrosslinks()
	for shard := range shardProcessed {
		if state.GetValidatorRegistryLastChangeSlot() >= crosslinks[shard].Slot {
			return false
		}
	}
	return true
}

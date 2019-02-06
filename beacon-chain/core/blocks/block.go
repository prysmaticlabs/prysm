// Package blocks contains block processing libraries. These libraries
// process and verify block specific messages such as PoW receipt root,
// RANDAO, validator deposits, exits and slashing proofs.
package blocks

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

var clock utils.Clock = &utils.RealClock{}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock(stateRoot []byte) *pb.BeaconBlock {
	block := &pb.BeaconBlock{
		Slot:               params.BeaconConfig().GenesisSlot,
		ParentRootHash32:   params.BeaconConfig().ZeroHash[:],
		StateRootHash32:    stateRoot,
		RandaoRevealHash32: params.BeaconConfig().ZeroHash[:],
		Signature:          params.BeaconConfig().EmptySignature,
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: []*pb.ProposerSlashing{},
			AttesterSlashings: []*pb.AttesterSlashing{},
			Attestations:      []*pb.Attestation{},
			Deposits:          []*pb.Deposit{},
			Exits:             []*pb.Exit{},
		},
	}
	return block
}

// IsRandaoValid verifies the validity of randao from block by comparing it with
// the proposer's randao from the beacon state.
func IsRandaoValid(blockRandao []byte, stateRandao []byte) bool {
	return hashutil.Hash(blockRandao) == bytesutil.ToBytes32(stateRandao)
}

// IsSlotValid compares the slot to the system clock to determine if the block is valid.
func IsSlotValid(slot uint64, genesisTime time.Time) bool {
	slotDuration := time.Duration(slot*params.BeaconConfig().SlotDuration) * time.Second
	validTimeThreshold := genesisTime.Add(slotDuration)
	return clock.Now().After(validTimeThreshold)
}

// BlockRoot returns the block root stored in the BeaconState for a given slot.
// It returns an error if the requested block root is not within the BeaconState.
// Spec pseudocode definition:
// 	def get_block_root(state: BeaconState, slot: int) -> Hash32:
//		"""
//		returns the block root at a recent ``slot``.
//		"""
//		assert state.slot <= slot + LATEST_BLOCK_ROOTS_LENGTH
//		assert slot < state.slot
//		return state.latest_block_roots[slot % LATEST_BLOCK_ROOTS_LENGTH]
func BlockRoot(state *pb.BeaconState, slot uint64) ([]byte, error) {
	//	Check to see if the requested block root lies within LatestBlockRootHash32S
	//	and if not generate error.
	var earliestSlot uint64
	var previousSlot uint64
	if state.Slot > uint64(len(state.LatestBlockRootHash32S)) {
		earliestSlot = state.Slot - uint64(len(state.LatestBlockRootHash32S))
	} else {
		earliestSlot = 0
	}
	previousSlot = state.Slot - 1
	if state.Slot > slot+uint64(len(state.LatestBlockRootHash32S)) || slot >= state.Slot {
		return []byte{}, fmt.Errorf("slot %d is not within expected range of %d to %d",
			slot,
			earliestSlot,
			previousSlot,
		)
	}
	return state.LatestBlockRootHash32S[slot%uint64(len(state.LatestBlockRootHash32S))], nil
}

// ProcessBlockRoots processes the previous block root into the state, by appending it
// to the most recent block roots.
// Spec:
//  Let previous_block_root be the tree_hash_root of the previous beacon block processed in the chain.
//	Set state.latest_block_roots[(state.slot - 1) % LATEST_BLOCK_ROOTS_LENGTH] = previous_block_root.
//	If state.slot % LATEST_BLOCK_ROOTS_LENGTH == 0 append merkle_root(state.latest_block_roots) to state.batched_block_roots.
func ProcessBlockRoots(state *pb.BeaconState, prevBlockRoot [32]byte) *pb.BeaconState {
	state.LatestBlockRootHash32S[(state.Slot-1)%params.BeaconConfig().LatestBlockRootsLength] = prevBlockRoot[:]
	if state.Slot%params.BeaconConfig().LatestBlockRootsLength == 0 {
		merkleRoot := hashutil.MerkleRoot(state.LatestBlockRootHash32S)
		state.BatchedBlockRootHash32S = append(state.BatchedBlockRootHash32S, merkleRoot)
	}
	return state
}

// EncodeDepositData converts a deposit input proto into an a byte slice
// of Simple Serialized deposit input followed by 8 bytes for a deposit value
// and 8 bytes for a unix timestamp, all in BigEndian format.
func EncodeDepositData(
	depositInput *pb.DepositInput,
	depositValue uint64,
	depositTimestamp int64,
) ([]byte, error) {
	wBuf := new(bytes.Buffer)
	if err := ssz.Encode(wBuf, depositInput); err != nil {
		return nil, fmt.Errorf("failed to encode deposit input: %v", err)
	}
	encodedInput := wBuf.Bytes()
	depositData := make([]byte, 0, 16+len(encodedInput))

	value := make([]byte, 8)
	binary.BigEndian.PutUint64(value, depositValue)

	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(depositTimestamp))

	depositData = append(depositData, value...)
	depositData = append(depositData, timestamp...)
	depositData = append(depositData, encodedInput...)

	return depositData, nil
}

// DecodeDepositInput unmarshalls a depositData byte slice into
// a proto *pb.DepositInput by using the Simple Serialize (SSZ)
// algorithm.
// TODO(#1253): Do not assume we will receive serialized proto objects - instead,
// replace completely by a common struct which can be simple serialized.
func DecodeDepositInput(depositData []byte) (*pb.DepositInput, error) {
	// Last 16 bytes of deposit data are 8 bytes for value
	// and 8 bytes for timestamp. Everything before that is a
	// Simple Serialized deposit input value.
	if len(depositData) < 16 {
		return nil, fmt.Errorf(
			"deposit data slice too small: len(depositData) = %d",
			len(depositData),
		)
	}
	depositInput := new(pb.DepositInput)
	// Since the value deposited and the timestamp are both 8 bytes each,
	// the deposit data is the chunk after the first 16 bytes.
	depositInputBytes := depositData[16:]
	rBuf := bytes.NewReader(depositInputBytes)
	if err := ssz.Decode(rBuf, depositInput); err != nil {
		return nil, fmt.Errorf("ssz decode failed: %v", err)
	}
	return depositInput, nil
}

// DecodeDepositAmountAndTimeStamp extracts the deposit amount and timestamp
// from the given deposit data.
func DecodeDepositAmountAndTimeStamp(depositData []byte) (uint64, int64, error) {
	// Last 16 bytes of deposit data are 8 bytes for value
	// and 8 bytes for timestamp. Everything before that is a
	// Simple Serialized deposit input value.
	if len(depositData) < 16 {
		return 0, 0, fmt.Errorf(
			"deposit data slice too small: len(depositData) = %d",
			len(depositData),
		)
	}

	// the amount occupies the first 8 bytes while the
	// timestamp occupies the next 8 bytes.
	amount := binary.BigEndian.Uint64(depositData[:8])
	timestamp := binary.BigEndian.Uint64(depositData[8:16])

	return amount, int64(timestamp), nil
}

// BlockChildren obtains the blocks in a list of observed blocks which have the current
// beacon block's hash as their parent root hash.
//
// Spec pseudocode definition:
//	Let get_children(store: Store, block: BeaconBlock) ->
//		List[BeaconBlock] returns the child blocks of the given block.
func BlockChildren(block *pb.BeaconBlock, observedBlocks []*pb.BeaconBlock) ([]*pb.BeaconBlock, error) {
	var children []*pb.BeaconBlock
	hash, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return nil, fmt.Errorf("could not hash block: %v", err)
	}
	for _, observed := range observedBlocks {
		if bytes.Equal(observed.ParentRootHash32, hash[:]) {
			children = append(children, observed)
		}
	}
	return children, nil
}

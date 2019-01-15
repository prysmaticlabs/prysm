// Package blocks contains block processing libraries. These libraries
// process and verify block specific messages such as PoW receipt root,
// RANDAO, validator deposits, exits and slashing proofs.
package blocks

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	bytesutil "github.com/prysmaticlabs/prysm/shared/bytes"
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
			CasperSlashings:   []*pb.CasperSlashing{},
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

// BlockRoot returns the block hash from input slot, the block hashes
// are stored in BeaconState.
//
// Spec pseudocode definition:
//   def get_block_root(state: BeaconState, slot: int) -> Hash32:
//     """
//     Returns the block hash at a recent ``slot``.
//     """
//     earliest_slot_in_array = state.slot - len(state.latest_block_roots)
//     assert earliest_slot_in_array <= slot < state.slot
//     return state.latest_block_roots[slot - earliest_slot_in_array]
func BlockRoot(state *pb.BeaconState, slot uint64) ([]byte, error) {
	var earliestSlot uint64

	// If the state slot is less than the length of state block root list, then
	// the earliestSlot would result in a negative number. Therefore we should
	// default earliestSlot = 0 in this case.
	if state.Slot > uint64(len(state.LatestBlockRootHash32S)) {
		earliestSlot = state.Slot - uint64(len(state.LatestBlockRootHash32S))
	}

	if slot < earliestSlot || slot >= state.Slot {
		return []byte{}, fmt.Errorf("slot %d out of bounds: %d <= slot < %d",
			slot,
			earliestSlot,
			state.Slot,
		)
	}

	return state.LatestBlockRootHash32S[slot-earliestSlot], nil
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

// ForkVersion Spec:
//	def get_fork_version(fork_data: ForkData,
//                     slot: int) -> int:
//    if slot < fork_data.fork_slot:
//        return fork_data.pre_fork_version
//    else:
//        return fork_data.post_fork_version
func ForkVersion(data *pb.ForkData, slot uint64) uint64 {
	if slot < data.ForkSlot {
		return data.PreForkVersion
	}
	return data.PostForkVersion
}

// DomainVersion Spec:
//	def get_domain(fork_data: ForkData,
//               slot: int,
//               domain_type: int) -> int:
//    return get_fork_version(
//        fork_data,
//        slot
//    ) * 2**32 + domain_type
func DomainVersion(data *pb.ForkData, slot uint64, domainType uint64) uint64 {
	constant := uint64(math.Pow(2, 32))
	return ForkVersion(data, slot)*constant + domainType
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

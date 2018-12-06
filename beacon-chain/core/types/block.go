package types

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var clock utils.Clock = &utils.RealClock{}

// Block defines a beacon chain core primitive.
type Block struct {
	data *pb.BeaconBlock
}

// NewBlock explicitly sets the data field of a block.
// Return block with default fields if data is nil.
func NewBlock(data *pb.BeaconBlock) *Block {
	if data == nil {
		var ancestorHashes = make([][]byte, 0, 32)
		// It is assumed when data==nil, a genesis block will be returned.
		return &Block{
			data: &pb.BeaconBlock{
				AncestorHash32S:               ancestorHashes,
				RandaoRevealHash32:            []byte{0},
				CandidatePowReceiptRootHash32: []byte{0},
				StateRootHash32:               []byte{0},
				Specials:                      []*pb.SpecialRecord{},
			},
		}
	}

	return &Block{data: data}
}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock(stateRoot [32]byte) *Block {
	// Genesis time here is static so error can be safely ignored.
	// #nosec G104
	protoGenesis, _ := ptypes.TimestampProto(params.BeaconConfig().GenesisTime)
	gb := NewBlock(nil)
	gb.data.Timestamp = protoGenesis
	gb.data.StateRootHash32 = stateRoot[:]
	return gb
}

// SlotNumber of the beacon block.
func (b *Block) SlotNumber() uint64 {
	return b.data.Slot
}

// ParentHash corresponding to parent beacon block.
func (b *Block) ParentHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.AncestorHash32S[0])
	return h
}

// Hash generates the blake2b hash of the block
func (b *Block) Hash() ([32]byte, error) {
	data, err := proto.Marshal(b.data)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not marshal block proto data: %v", err)
	}
	return hashutil.Hash(data), nil
}

// Proto returns the underlying protobuf data within a block primitive.
func (b *Block) Proto() *pb.BeaconBlock {
	return b.data
}

// Marshal encodes block object into the wire format.
func (b *Block) Marshal() ([]byte, error) {
	return proto.Marshal(b.data)
}

// Timestamp returns the Go type time.Time from the protobuf type contained in the block.
func (b *Block) Timestamp() (time.Time, error) {
	return ptypes.Timestamp(b.data.Timestamp)
}

// AncestorHash32S of the block.
func (b *Block) AncestorHash32S() [][]byte {
	return b.data.AncestorHash32S
}

// AttestationCount returns the number of attestations.
func (b *Block) AttestationCount() int {
	return len(b.data.Attestations)
}

// Attestations returns an array of attestations in the block.
func (b *Block) Attestations() []*pb.AggregatedAttestation {
	return b.data.Attestations
}

// CandidatePowReceiptRootHash32 returns a keccak256 hash corresponding to a PoW chain block.
func (b *Block) CandidatePowReceiptRootHash32() common.Hash {
	return common.BytesToHash(b.data.CandidatePowReceiptRootHash32)
}

// RandaoRevealHash32 returns the blake2b randao hash.
func (b *Block) RandaoRevealHash32() [32]byte {
	var h [32]byte
	copy(h[:], b.data.RandaoRevealHash32)
	return h
}

// StateRootHash32 returns the state hash.
func (b *Block) StateRootHash32() [32]byte {
	var h [32]byte
	copy(h[:], b.data.StateRootHash32)
	return h
}

// IsRandaoValid verifies the validity of randao from block by comparing it with
// the proposer's randao from the beacon state.
func (b *Block) IsRandaoValid(stateRandao []byte) bool {
	var h [32]byte
	copy(h[:], stateRandao)
	blockRandaoRevealHash32 := b.RandaoRevealHash32()
	return hashutil.Hash(blockRandaoRevealHash32[:]) == h
}

// IsSlotValid compares the slot to the system clock to determine if the block is valid.
func (b *Block) IsSlotValid(genesisTime time.Time) bool {
	slotDuration := time.Duration(b.SlotNumber()*params.BeaconConfig().SlotDuration) * time.Second
	validTimeThreshold := genesisTime.Add(slotDuration)
	return clock.Now().After(validTimeThreshold)
}

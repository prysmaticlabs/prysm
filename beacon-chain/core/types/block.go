package types

import (
	"fmt"
	"time"

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
				AncestorHashes: ancestorHashes,
				RandaoReveal:   []byte{0},
				StateRoot:      []byte{0},
				Specials:       []*pb.SpecialRecord{},
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
	gb.data.StateRoot = stateRoot[:]
	return gb
}

// SlotNumber of the beacon block.
func (b *Block) SlotNumber() uint64 {
	return b.data.Slot
}

// ParentHash corresponding to parent beacon block.
func (b *Block) ParentHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.AncestorHashes[0])
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

// AncestorHashes of the block.
func (b *Block) AncestorHashes() [][]byte {
	return b.data.AncestorHashes
}

// AttestationCount returns the number of attestations.
func (b *Block) AttestationCount() int {
	return len(b.data.Attestations)
}

// Attestations returns an array of attestations in the block.
func (b *Block) Attestations() []*pb.AttestationRecord {
	return b.data.Attestations
}

// RandaoReveal returns the blake2b randao hash.
func (b *Block) RandaoReveal() [32]byte {
	var h [32]byte
	copy(h[:], b.data.RandaoReveal)
	return h
}

// StateRoot returns the state hash.
func (b *Block) StateRoot() [32]byte {
	var h [32]byte
	copy(h[:], b.data.StateRoot)
	return h
}

// IsRandaoValid verifies the validity of randao from block by comparing it with
// the proposer's randao from the beacon state.
func (b *Block) IsRandaoValid(stateRandao []byte) bool {
	var h [32]byte
	copy(h[:], stateRandao)
	blockRandaoReveal := b.RandaoReveal()
	return hashutil.Hash(blockRandaoReveal[:]) == h
}

// IsSlotValid compares the slot to the system clock to determine if the block is valid.
func (b *Block) IsSlotValid(genesisTime time.Time) bool {
	slotDuration := time.Duration(b.SlotNumber()*params.BeaconConfig().SlotDuration) * time.Second
	validTimeThreshold := genesisTime.Add(slotDuration)
	return clock.Now().After(validTimeThreshold)
}

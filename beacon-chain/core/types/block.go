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
				AncestorHashes: ancestorHashes,
				RandaoReveal:   []byte{0},
				PowChainRef:    []byte{0},
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

// AttestationCount returns the number of attestations.
func (b *Block) AttestationCount() int {
	return len(b.data.Attestations)
}

// Attestations returns an array of attestations in the block.
func (b *Block) Attestations() []*pb.AggregatedAttestation {
	return b.data.Attestations
}

// PowChainRef returns a keccak256 hash corresponding to a PoW chain block.
func (b *Block) PowChainRef() common.Hash {
	return common.BytesToHash(b.data.PowChainRef)
}

// StateRoot returns the state hash.
func (b *Block) StateRoot() [32]byte {
	var h [32]byte
	copy(h[:], b.data.StateRoot)
	return h
}

// isSlotValid compares the slot to the system clock to determine if the block is valid.
func (b *Block) isSlotValid(genesisTime time.Time) bool {
	slotDuration := time.Duration(b.SlotNumber()*params.BeaconConfig().SlotDuration) * time.Second
	validTimeThreshold := genesisTime.Add(slotDuration)
	return clock.Now().After(validTimeThreshold)
}

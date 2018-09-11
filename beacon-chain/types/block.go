// Package types defines the essential types used throughout the beacon-chain.
package types

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/blake2b"
)

var log = logrus.WithField("prefix", "types")

var genesisTime = time.Date(2018, 9, 0, 0, 0, 0, 0, time.UTC) // September 2019
var clock utils.Clock = &utils.RealClock{}

// Block defines a beacon chain core primitive.
type Block struct {
	data *pb.BeaconBlock
}

// NewBlock explicitly sets the data field of a block.
// Return block with default fields if data is nil.
func NewBlock(data *pb.BeaconBlock) *Block {
	if data == nil {
		return &Block{
			data: &pb.BeaconBlock{
				ParentHash:            []byte{0},
				SlotNumber:            0,
				RandaoReveal:          []byte{0},
				Attestations:          []*pb.AttestationRecord{},
				PowChainRef:           []byte{0},
				ActiveStateHash:       []byte{0},
				CrystallizedStateHash: []byte{0},
				// NOTE: this field only exists to determine the timestamp of the genesis block.
				// As of the v2.1 spec, the timestamp of blocks after genesis are not used.
				Timestamp: ptypes.TimestampNow(),
			},
		}
	}

	return &Block{data: data}
}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
//
// TODO(#495): Add more default fields.
func NewGenesisBlock() (*Block, error) {
	protoGenesis, err := ptypes.TimestampProto(genesisTime)
	if err != nil {
		return nil, err
	}
	return &Block{
		data: &pb.BeaconBlock{
			Timestamp:  protoGenesis,
			ParentHash: []byte{},
		},
	}, nil
}

// Proto returns the underlying protobuf data within a block primitive.
func (b *Block) Proto() *pb.BeaconBlock {
	return b.data
}

// Marshal encodes block object into the wire format.
func (b *Block) Marshal() ([]byte, error) {
	return proto.Marshal(b.data)
}

// Hash generates the blake2b hash of the block
func (b *Block) Hash() ([32]byte, error) {
	data, err := proto.Marshal(b.data)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not marshal block proto data: %v", err)
	}
	var hash [32]byte
	h := blake2b.Sum512(data)
	copy(hash[:], h[:32])
	return hash, nil
}

// ParentHash corresponding to parent beacon block.
func (b *Block) ParentHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.ParentHash)
	return h
}

// SlotNumber of the beacon block.
func (b *Block) SlotNumber() uint64 {
	return b.data.SlotNumber
}

// PowChainRef returns a keccak256 hash corresponding to a PoW chain block.
func (b *Block) PowChainRef() common.Hash {
	return common.BytesToHash(b.data.PowChainRef)
}

// RandaoReveal returns the blake2b randao hash.
func (b *Block) RandaoReveal() [32]byte {
	var h [32]byte
	copy(h[:], b.data.RandaoReveal)
	return h
}

// ActiveStateHash returns the active state hash.
func (b *Block) ActiveStateHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.ActiveStateHash)
	return h
}

// CrystallizedStateHash returns the crystallized state hash.
func (b *Block) CrystallizedStateHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.CrystallizedStateHash)
	return h
}

// AttestationCount returns the number of attestations.
func (b *Block) AttestationCount() int {
	return len(b.data.Attestations)
}

// Attestations returns an array of attestations in the block.
func (b *Block) Attestations() []*pb.AttestationRecord {
	return b.data.Attestations
}

// Timestamp returns the Go type time.Time from the protobuf type contained in the block.
func (b *Block) Timestamp() (time.Time, error) {
	return ptypes.Timestamp(b.data.Timestamp)
}

// isSlotValid compares the slot to the system clock to determine if the block is valid.
func (b *Block) isSlotValid() bool {
	slotDuration := time.Duration(b.SlotNumber()*params.SlotDuration) * time.Second
	validTimeThreshold := genesisTime.Add(slotDuration)

	return clock.Now().After(validTimeThreshold)
}

// IsValid is called to decide if an incoming p2p block can be processed.
// It checks the slot against the system clock, and the validity of the included attestations.
// Existence of the parent block and the PoW chain block is checked outside of this function because they require additional dependencies.
func (b *Block) IsValid(aState *ActiveState, cState *CrystallizedState) bool {
	_, err := b.Hash()
	if err != nil {
		log.Debugf("Could not hash incoming block: %v", err)
		return false
	}

	if b.SlotNumber() == 0 {
		log.Debug("Can not process a genesis block")
		return false
	}

	if !b.isSlotValid() {
		log.Debugf("slot of block is too high: %d", b.SlotNumber())
		return false
	}

	for index, attestation := range b.Attestations() {
		if !b.isAttestationValid(index, aState, cState) {
			log.Debugf("attestation invalid: %v", attestation)
			return false
		}
	}

	return true
}

// isAttestationValid validates an attestation in a block.
// Attestations are cross-checked against validators in CrystallizedState.ShardAndCommitteesForSlots.
// In addition, the signature is verified by constructing the list of parent hashes using ActiveState.RecentBlockHashes.
func (b *Block) isAttestationValid(attestationIndex int, aState *ActiveState, cState *CrystallizedState) bool {
	// Validate attestation's slot number has is within range of incoming block number.
	slotNumber := b.SlotNumber()
	attestation := b.Attestations()[attestationIndex]
	if attestation.Slot > slotNumber {
		log.Debugf("attestation slot number can't be higher than block slot number. Found: %d, Needed lower than: %d",
			attestation.Slot,
			slotNumber)
		return false
	}
	if int(attestation.Slot) < int(slotNumber)-params.CycleLength {
		log.Debugf("attestation slot number can't be lower than block slot number by one CycleLength. Found: %v, Needed greater than: %v",
			attestation.Slot,
			slotNumber-params.CycleLength)
		return false
	}

	if attestation.JustifiedSlot > cState.LastJustifiedSlot() {
		log.Debugf("attestation's last justified slot has to match crystallied state's last justified slot. Found: %d. Want: %d",
			attestation.JustifiedSlot,
			cState.LastJustifiedSlot())
		return false
	}

	// TODO(#468): Validate last justified block hash matches in the crystallizedState.

	// Get all the block hashes up to cycle length.
	parentHashes := aState.getSignedParentHashes(b, attestation)
	attesterIndices, err := cState.getAttesterIndices(attestation)
	if err != nil {
		log.Debugf("unable to get validator committee: %v", attesterIndices)
		return false
	}

	// Verify attester bitfields matches crystallized state's prev computed bitfield.
	if !casper.AreAttesterBitfieldsValid(attestation, attesterIndices) {
		return false
	}

	// TODO(#258): Generate validators aggregated pub key.

	// Hash parentHashes + shardID + slotNumber + shardBlockHash into a message to use to
	// to verify with aggregated public key and aggregated attestation signature.
	msg := make([]byte, binary.MaxVarintLen64)
	var signedHashesStr []byte
	for _, parentHash := range parentHashes {
		signedHashesStr = append(signedHashesStr, parentHash[:]...)
		signedHashesStr = append(signedHashesStr, byte(' '))
	}
	binary.PutUvarint(msg, attestation.Slot%params.CycleLength)
	msg = append(msg, signedHashesStr...)
	binary.PutUvarint(msg, attestation.ShardId)
	msg = append(msg, attestation.ShardBlockHash...)

	msgHash := blake2b.Sum512(msg)

	log.Debugf("Attestation message for shard: %v, slot %v, block hash %v is: %v",
		attestation.ShardId, attestation.Slot, attestation.ShardBlockHash, msgHash)

	// TODO(#258): Verify msgHash against aggregated pub key and aggregated signature.
	return true
}

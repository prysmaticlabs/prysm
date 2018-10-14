// Package types defines the essential types used throughout the beacon-chain.
package types

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "types")

var clock utils.Clock = &utils.RealClock{}

// Block defines a beacon chain core primitive.
type Block struct {
	data *pb.BeaconBlock
}

type beaconDB interface {
	HasBlock(h [32]byte) (bool, error)
}

// NewBlock explicitly sets the data field of a block.
// Return block with default fields if data is nil.
func NewBlock(data *pb.BeaconBlock) *Block {
	if data == nil {
		var ancestorHashes = make([][]byte, 0, 32)

		//It is assumed when data==nil, you're asking for a Genesis Block
		return &Block{
			data: &pb.BeaconBlock{
				AncestorHashes:        ancestorHashes,
				RandaoReveal:          []byte{0},
				PowChainRef:           []byte{0},
				ActiveStateRoot:       []byte{0},
				CrystallizedStateRoot: []byte{0},
			},
		}
	}

	return &Block{data: data}
}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock(activeStateRoot [32]byte, crystallizedStateRoot [32]byte) *Block {
	// Genesis time here is static so error can be safely ignored.
	// #nosec G104
	protoGenesis, _ := ptypes.TimestampProto(params.GetConfig().GenesisTime)
	gb := NewBlock(nil)
	gb.data.Timestamp = protoGenesis

	gb.data.ActiveStateRoot = activeStateRoot[:]
	gb.data.CrystallizedStateRoot = crystallizedStateRoot[:]
	return gb
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
	return hashutil.Hash(data), nil
}

// ParentHash corresponding to parent beacon block.
func (b *Block) ParentHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.AncestorHashes[0])
	return h
}

// SlotNumber of the beacon block.
func (b *Block) SlotNumber() uint64 {
	return b.data.Slot
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

// ActiveStateRoot returns the active state hash.
func (b *Block) ActiveStateRoot() [32]byte {
	var h [32]byte
	copy(h[:], b.data.ActiveStateRoot)
	return h
}

// CrystallizedStateRoot returns the crystallized state hash.
func (b *Block) CrystallizedStateRoot() [32]byte {
	var h [32]byte
	copy(h[:], b.data.CrystallizedStateRoot)
	return h
}

// AttestationCount returns the number of attestations.
func (b *Block) AttestationCount() int {
	return len(b.data.Attestations)
}

// Attestations returns an array of attestations in the block.
func (b *Block) Attestations() []*pb.AggregatedAttestation {
	return b.data.Attestations
}

// Timestamp returns the Go type time.Time from the protobuf type contained in the block.
func (b *Block) Timestamp() (time.Time, error) {
	return ptypes.Timestamp(b.data.Timestamp)
}

// isSlotValid compares the slot to the system clock to determine if the block is valid.
func (b *Block) isSlotValid(genesisTime time.Time) bool {
	slotDuration := time.Duration(b.SlotNumber()*params.GetConfig().SlotDuration) * time.Second
	validTimeThreshold := genesisTime.Add(slotDuration)
	return clock.Now().After(validTimeThreshold)
}

// IsValid is called to decide if an incoming p2p block can be processed. It checks for following conditions:
// 1.) Ensure local time is large enough to process this block's slot.
// 2.) Verify that the parent block's proposer's attestation is included.
func (b *Block) IsValid(
	db beaconDB,
	aState *ActiveState,
	cState *CrystallizedState,
	parentSlot uint64,
	enableAttestationValidity bool,
	genesisTime time.Time) bool {
	_, err := b.Hash()
	if err != nil {
		log.Errorf("Could not hash incoming block: %v", err)
		return false
	}

	if b.SlotNumber() == 0 {
		log.Error("Can not process a genesis block")
		return false
	}

	if !b.isSlotValid(genesisTime) {
		log.Errorf("Slot of block is too high: %d", b.SlotNumber())
		return false
	}

	if enableAttestationValidity {
		// verify proposer from last slot is in the first attestation object in AggregatedAttestation.
		_, proposerIndex, err := casper.ProposerShardAndIndex(
			cState.ShardAndCommitteesForSlots(),
			cState.LastStateRecalculationSlot(),
			parentSlot)
		if err != nil {
			log.Errorf("Can not get proposer index %v", err)
			return false
		}
		log.Infof("Proposer index: %v", proposerIndex)
		if !bitutil.CheckBit(b.Attestations()[0].AttesterBitfield, int(proposerIndex)) {
			log.Errorf("Can not locate proposer in the first attestation of AttestionRecord %v", err)
			return false
		}

		for index, attestation := range b.Attestations() {
			if !b.isAttestationValid(index, db, aState, cState, parentSlot) {
				log.Debugf("attestation invalid: %v", attestation)
				return false
			}
		}
	}

	return true
}

// isAttestationValid validates an attestation in a block.
// Attestations are cross-checked against validators in CrystallizedState.ShardAndCommitteesForSlots.
// In addition, the signature is verified by constructing the list of parent hashes using ActiveState.RecentBlockHashes.
func (b *Block) isAttestationValid(attestationIndex int, db beaconDB, aState *ActiveState, cState *CrystallizedState, parentSlot uint64) bool {
	// Validate attestation's slot number has is within range of incoming block number.
	attestation := b.Attestations()[attestationIndex]
	if !isAttestationSlotNumberValid(attestation.Slot, parentSlot) {
		return false
	}

	if attestation.JustifiedSlot > cState.LastJustifiedSlot() {
		log.Debugf("attestation's justified slot has to be earlier or equal to crystallized state's last justified slot. Found: %d. Want <=: %d",
			attestation.JustifiedSlot,
			cState.LastJustifiedSlot())
		return false
	}

	hash := [32]byte{}
	copy(hash[:], attestation.JustifiedBlockHash)
	blockInChain, err := db.HasBlock(hash)
	if err != nil {
		log.Errorf("unable to determine if attestation justified block is in the DB: %s", err)
		return false
	}

	if !blockInChain {
		log.Debugf("The attestion's justifed block hash has to be in the current chain, but was not found.  Justified block hash: %v",
			attestation.JustifiedBlockHash)
		return false
	}

	// Get all the block hashes up to cycle length.
	parentHashes := aState.getSignedParentHashes(b, attestation)
	attesterIndices, err := cState.getAttesterIndices(attestation)
	if err != nil {
		log.Debugf("Unable to get validator committee: %v", attesterIndices)
		return false
	}

	// Verify attester bitfields matches crystallized state's prev computed bitfield.
	if !casper.AreAttesterBitfieldsValid(attestation, attesterIndices) {
		return false
	}

	// TODO(#258): Generate validators aggregated pub key.

	attestationMsg := AttestationMsg(
		parentHashes,
		attestation.ShardBlockHash,
		attestation.Slot,
		attestation.Shard,
		attestation.JustifiedSlot)

	log.Debugf("Attestation message for shard: %v, slot %v, block hash %v is: %v",
		attestation.Shard, attestation.Slot, attestation.ShardBlockHash, attestationMsg)

	// TODO(#258): Verify msgHash against aggregated pub key and aggregated signature.
	return true
}

func isAttestationSlotNumberValid(attestationSlot uint64, parentSlot uint64) bool {
	if parentSlot != 0 && attestationSlot > parentSlot {
		log.Debugf("attestation slot number can't be higher than parent block's slot number. Found: %d, Needed lower than: %d",
			attestationSlot,
			parentSlot)
		return false
	}

	if parentSlot >= params.GetConfig().CycleLength-1 && attestationSlot < parentSlot-params.GetConfig().CycleLength+1 {
		log.Debugf("attestation slot number can't be lower than parent block's slot number by one CycleLength. Found: %d, Needed greater than: %d",
			attestationSlot,
			parentSlot-params.GetConfig().CycleLength+1)
		return false
	}

	return true
}

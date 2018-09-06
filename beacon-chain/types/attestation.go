// Package types defines the essential types used throughout the beacon-chain.
package types

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"golang.org/x/crypto/blake2b"
)

// Attestation is the primary source of load on the beacon chain, it's used to
// attest to some parent block in the chain and block hash in a shard.
type Attestation struct {
	data *pb.AttestationRecord
}

// NewAttestation explicitly sets the data field of a attestation.
// Return attestation with default fields if data is nil.
func NewAttestation(data *pb.AttestationRecord) *Attestation {
	if data == nil {
		return &Attestation{
			data: &pb.AttestationRecord{
				Slot:                0,
				ShardId:             0,
				JustifiedSlot:       0,
				JustifiedBlockHash:  []byte{},
				ShardBlockHash:      []byte{},
				AttesterBitfield:    []byte{},
				ObliqueParentHashes: [][]byte{{}},
				AggregateSig:        []uint64{},
			},
		}
	}
	return &Attestation{data: data}
}

// Proto returns the underlying protobuf data.
func (a *Attestation) Proto() *pb.AttestationRecord {
	return a.data
}

// Marshal encodes block object into the wire format.
func (a *Attestation) Marshal() ([]byte, error) {
	return proto.Marshal(a.data)
}

// Hash generates the blake2b hash of the attestation.
func (a *Attestation) Hash() ([32]byte, error) {
	data, err := proto.Marshal(a.data)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not marshal attestation proto data: %v", err)
	}
	var hash [32]byte
	h := blake2b.Sum512(data)
	copy(hash[:], h[:32])
	return hash, nil
}

// SlotNumber of the block, which this attestation is attesting to.
func (a *Attestation) SlotNumber() uint64 {
	return a.data.Slot
}

// ShardID of the block, which this attestation is attesting to.
func (a *Attestation) ShardID() uint64 {
	return a.data.ShardId
}

// JustifiedSlotNumber of the attestation should be earlier than the last justified slot in crystallized state.
func (a *Attestation) JustifiedSlotNumber() uint64 {
	return a.data.Slot
}

// JustifiedBlockHash should be in the chain of the block being processed.
func (a *Attestation) JustifiedBlockHash() []byte {
	return a.data.JustifiedBlockHash
}

// AttesterBitfield represents which validator in the committee has voted.
func (a *Attestation) AttesterBitfield() []byte {
	return a.data.AttesterBitfield
}

// ObliqueParentHashes represents the block hashes this attestation is not attesting for.
func (a *Attestation) ObliqueParentHashes() [][32]byte {
	var obliqueParentHashes [][32]byte
	for _, hash := range a.data.ObliqueParentHashes {
		var h [32]byte
		copy(h[:], hash)
		obliqueParentHashes = append(obliqueParentHashes, h)
	}
	return obliqueParentHashes
}

// AggregateSig represents the aggregated signature from all the validators attesting to this block.
func (a *Attestation) AggregateSig() []uint64 {
	return a.data.AggregateSig
}

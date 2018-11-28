package types

import (
	"encoding/binary"
	"fmt"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Attestation is the primary source of load on the beacon chain, it's used to
// attest to some parent block in the chain and block hash in a shard.
type Attestation struct {
	data *pb.AggregatedAttestation
}

// NewAttestation explicitly sets the data field of a attestation.
// Return attestation with default fields if data is nil.
func NewAttestation(data *pb.AggregatedAttestation) *Attestation {
	if data == nil {
		return &Attestation{
			data: &pb.AggregatedAttestation{
				AttesterBitfield: []byte{},
				AggregateSig:     []uint64{},
				PocBitfield:      []byte{},
				SignedData: &pb.AttestationSignedData{
					Slot:               0,
					Shard:              0,
					BlockHash:          []byte{},
					CycleBoundaryHash:  []byte{},
					LastCrosslinkHash:  []byte{},
					JustifiedSlot:      0,
					JustifiedBlockHash: []byte{},
					ShardBlockHash:     []byte{},
				},
			},
		}
	}
	return &Attestation{data: data}
}

// AttestationMsg hashes shardID + slotNumber + shardBlockHash + justifiedSlot
// into a message to use for verifying with aggregated public key and signature.
func AttestationMsg(
	blockHash []byte,
	slot uint64,
	shardID uint64,
	justifiedSlot uint64,
	forkVersion uint64,
) [32]byte {
	msg := make([]byte, binary.MaxVarintLen64)
	binary.BigEndian.PutUint64(msg, forkVersion)
	binary.PutUvarint(msg, slot%params.BeaconConfig().CycleLength)
	binary.PutUvarint(msg, shardID)
	msg = append(msg, blockHash...)
	binary.PutUvarint(msg, justifiedSlot)
	return hashutil.Hash(msg)
}

// Proto returns the underlying protobuf data.
func (a *Attestation) Proto() *pb.AggregatedAttestation {
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
	return hashutil.Hash(data), nil
}

// SignedData returns the innter attestation signed data object.
func (a *Attestation) SignedData() *pb.AttestationSignedData {
	return a.data.SignedData
}

// Key generates the blake2b hash of the following attestation fields:
// slotNumber + shardID + blockHash
// This is used for attestation table look up in localDB.
func (a *Attestation) Key() [32]byte {
	key := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(key, a.SlotNumber())
	binary.PutUvarint(key, a.ShardID())
	key = append(key, a.ShardBlockHash()...)
	return hashutil.Hash(key)
}

// SlotNumber of the block, which this attestation is attesting to.
func (a *Attestation) SlotNumber() uint64 {
	return a.data.SignedData.Slot
}

// ShardID of the block, which this attestation is attesting to.
func (a *Attestation) ShardID() uint64 {
	return a.data.SignedData.Shard
}

// ShardBlockHash of the block, which this attestation is attesting to.
func (a *Attestation) ShardBlockHash() []byte {
	return a.data.SignedData.ShardBlockHash
}

// JustifiedSlotNumber of the attestation should be earlier than the last justified slot in crystallized state.
func (a *Attestation) JustifiedSlotNumber() uint64 {
	return a.data.SignedData.JustifiedSlot
}

// JustifiedBlockHash should be in the chain of the block being processed.
func (a *Attestation) JustifiedBlockHash() []byte {
	return a.data.SignedData.JustifiedBlockHash
}

// AttesterBitfield represents which validator in the committee has voted.
func (a *Attestation) AttesterBitfield() []byte {
	return a.data.AttesterBitfield
}

// AggregateSig represents the aggregated signature from all the validators attesting to this block.
func (a *Attestation) AggregateSig() []uint64 {
	return a.data.AggregateSig
}

// VerifyProposerAttestation verifies the proposer's attestation of the block.
// Proposers broadcast the attestation along with the block to its peers.
func (a *Attestation) VerifyProposerAttestation(pubKey [32]byte, proposerShardID uint64) error {
	// Verify the attestation attached with block response.
	// Get proposer index and shardID.
	attestationMsg := AttestationMsg(
		a.ShardBlockHash(),
		a.SlotNumber(),
		proposerShardID,
		a.JustifiedSlotNumber(),
		params.BeaconConfig().InitialForkVersion,
	)
	_ = attestationMsg
	_ = pubKey
	// TODO(#258): use attestationMsg to verify against signature and public key. Return error if incorrect.
	return nil
}

// ContainsValidator checks if the validator is included in the attestation.
// TODO(#569): Modify method to accept a single index rather than a bitfield.
func (a *Attestation) ContainsValidator(bitfield []byte) bool {
	savedAttestationBitfield := a.AttesterBitfield()
	for i := 0; i < len(bitfield); i++ {
		if bitfield[i]&savedAttestationBitfield[i] != 0 {
			return true
		}
	}
	return false
}

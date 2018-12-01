package types

import (
	"encoding/binary"
	"fmt"

	"github.com/golang/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// AggregatedAttestation is the primary source of load on the beacon chain, it's used to
// attest to some parent block in the chain and block hash in a shard.
type AggregatedAttestation struct {
	data *pb.AggregatedAttestation
}

// ProcessedAttestation simply tracks slot inclusion and does not contain an aggregate
// signature value from validators.
type ProcessedAttestation struct {
	data *pb.ProcessedAttestation
}

// NewAggregatedAttestation explicitly sets the data field of an
// attestation record including an aggregate BLS signature.
// This returns an attestation with default fields if data is nil.
func NewAggregatedAttestation(data *pb.AggregatedAttestation) *AggregatedAttestation {
	if data == nil {
		return &AggregatedAttestation{
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
	return &AggregatedAttestation{data: data}
}

// NewProcessedAttestation creates an attestation record instance that
// does not care about aggregate signatures and just tracks the
// slot it was included.
func NewProcessedAttestation(data *pb.ProcessedAttestation) *ProcessedAttestation {
	if data == nil {
		return &ProcessedAttestation{
			data: &pb.ProcessedAttestation{
				AttesterBitfield: []byte{},
				PocBitfield:      []byte{},
				SlotIncluded:     0,
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
	return &ProcessedAttestation{data: data}
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
func (a *AggregatedAttestation) Proto() *pb.AggregatedAttestation {
	return a.data
}

// Marshal encodes block object into the wire format.
func (a *AggregatedAttestation) Marshal() ([]byte, error) {
	return proto.Marshal(a.data)
}

// Hash generates the blake2b hash of the attestation.
func (a *AggregatedAttestation) Hash() ([32]byte, error) {
	data, err := proto.Marshal(a.data)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not marshal attestation proto data: %v", err)
	}
	return hashutil.Hash(data), nil
}

// SignedData returns the innter attestation signed data object.
func (a *AggregatedAttestation) SignedData() *pb.AttestationSignedData {
	return a.data.SignedData
}

// Proto returns the underlying protobuf data.
func (a *ProcessedAttestation) Proto() *pb.ProcessedAttestation {
	return a.data
}

// Marshal encodes block object into the wire format.
func (a *ProcessedAttestation) Marshal() ([]byte, error) {
	return proto.Marshal(a.data)
}

// Hash generates the blake2b hash of the attestation.
func (a *ProcessedAttestation) Hash() ([32]byte, error) {
	data, err := proto.Marshal(a.data)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not marshal attestation proto data: %v", err)
	}
	return hashutil.Hash(data), nil
}

// SignedData returns the inner attestation signed data object.
func (a *ProcessedAttestation) SignedData() *pb.AttestationSignedData {
	return a.data.SignedData
}

// SlotIncluded returns the inner proto slot included field.
func (a *ProcessedAttestation) SlotIncluded() uint64 {
	return a.data.SlotIncluded
}

// AttestationKey generates the blake2b hash of the following attestation fields:
// slotNumber + shardID + blockHash
// This is used for attestation table look up in localDB.
func AttestationKey(attSigned *pb.AttestationSignedData) [32]byte {
	key := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(key, attSigned.GetSlot())
	binary.PutUvarint(key, attSigned.GetShard())
	key = append(key, attSigned.GetShardBlockHash()...)
	return hashutil.Hash(key)
}

// AttesterBitfield represents which validator in the committee has voted.
func (a *AggregatedAttestation) AttesterBitfield() []byte {
	return a.data.AttesterBitfield
}

// AggregateSig represents the aggregated signature from all the validators attesting to this block.
func (a *AggregatedAttestation) AggregateSig() []uint64 {
	return a.data.AggregateSig
}

// VerifyProposerAttestation verifies the proposer's attestation of the block.
// Proposers broadcast the attestation along with the block to its peers.
func VerifyProposerAttestation(attSigned *pb.AttestationSignedData, pubKey [32]byte, proposerShardID uint64) error {
	// Verify the attestation attached with block response.
	// Get proposer index and shardID.
	attestationMsg := AttestationMsg(
		attSigned.GetShardBlockHash(),
		attSigned.GetSlot(),
		proposerShardID,
		attSigned.GetJustifiedSlot(),
		params.BeaconConfig().InitialForkVersion,
	)
	_ = attestationMsg
	_ = pubKey
	// TODO(#258): use attestationMsg to verify against signature and public key. Return error if incorrect.
	return nil
}

// ContainsValidator checks if the validator is included in the attestation.
// TODO(#569): Modify method to accept a single index rather than a bitfield.
func ContainsValidator(attesterBitfield []byte, bitfield []byte) bool {
	for i := 0; i < len(bitfield); i++ {
		if bitfield[i]&attesterBitfield[i] != 0 {
			return true
		}
	}
	return false
}

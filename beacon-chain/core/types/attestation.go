package types

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessedAttestation tracks the inclusion slot of the attestation
// in the beacon chain.
type ProcessedAttestation struct {
	data *pb.ProcessedAttestation
}

// AggregatedAttestation contains an aggregate signature from validators in
// the beacon chain's state.
type AggregatedAttestation struct {
	data *pb.AggregatedAttestation
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
// slot it was included in.
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
func (agg *AggregatedAttestation) Proto() *pb.AggregatedAttestation {
	return agg.data
}

// Proto returns the underlying protobuf data.
func (proc *ProcessedAttestation) Proto() *pb.ProcessedAttestation {
	return proc.data
}

// Marshal encodes object into the wire format.
func (agg *AggregatedAttestation) Marshal() ([]byte, error) {
	return proto.Marshal(agg.data)
}

// Marshal encodes object into the wire format.
func (proc *ProcessedAttestation) Marshal() ([]byte, error) {
	return proto.Marshal(proc.data)
}

// Hash generates the blake2b hash of the attestation.
func (agg *AggregatedAttestation) Hash() ([32]byte, error) {
	data, err := proto.Marshal(agg.data)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not marshal attestation proto data: %v", err)
	}
	return hashutil.Hash(data), nil
}

// Hash generates the blake2b hash of the attestation.
func (proc *ProcessedAttestation) Hash() ([32]byte, error) {
	data, err := proto.Marshal(proc.data)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not marshal attestation proto data: %v", err)
	}
	return hashutil.Hash(data), nil
}

// SignedData returns the innter attestation signed data object.
func (agg *AggregatedAttestation) SignedData() *pb.AttestationSignedData {
	return agg.data.SignedData
}

// SignedData returns the innter attestation signed data object.
func (proc *ProcessedAttestation) SignedData() *pb.AttestationSignedData {
	return proc.data.SignedData
}

// AttestationKey generates the blake2b hash of the following attestation fields:
// slotNumber + shardID + blockHash
// This is used for attestation table look up in localDB.
func AttestationKey(slotNumber uint64, shardID uint64, shardBlockHash []byte) [32]byte {
	key := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(key, slotNumber)
	binary.PutUvarint(key, shardID)
	key = append(key, shardBlockHash...)
	return hashutil.Hash(key)
}

// AttesterBitfield represents which validator in the committee has voted.
func (agg *AggregatedAttestation) AttesterBitfield() []byte {
	return agg.data.AttesterBitfield
}

// AttesterBitfield represents which validator in the committee has voted.
func (proc *ProcessedAttestation) AttesterBitfield() []byte {
	return proc.data.AttesterBitfield
}

// SlotIncluded represents the slot the processed attestation was added to the chain.
func (proc *ProcessedAttestation) SlotIncluded() uint64 {
	return proc.data.SlotIncluded
}

// AggregateSig represents the aggregated signature from all the validators attesting to a block.
func (agg *AggregatedAttestation) AggregateSig() []uint64 {
	return agg.data.AggregateSig
}

// VerifyProposerAttestation verifies the proposer's attestation of the block.
// Proposers broadcast the attestation along with the block to its peers.
func VerifyProposerAttestation(
	pubKey [32]byte,
	proposerShardID uint64,
	shardBlockHash []byte,
	slotNumber uint64,
	justifiedSlotNumber uint64,
) error {
	// Verify the attestation attached with block response.
	// Get proposer index and shardID.
	attestationMsg := AttestationMsg(
		shardBlockHash,
		slotNumber,
		proposerShardID,
		justifiedSlotNumber,
		params.BeaconConfig().InitialForkVersion,
	)
	_ = attestationMsg
	_ = pubKey
	// TODO(#258): use attestationMsg to verify against signature and public key. Return error if incorrect.
	return nil
}

// ContainsValidator checks if the validator is included in the attestation.
func ContainsValidator(attesterBitfield []byte, bitfield []byte) bool {
	for i := 0; i < len(bitfield); i++ {
		log.Printf("%v", bitfield[i]&attesterBitfield[i])
		if bitfield[i]&attesterBitfield[i] != 0 {
			return true
		}
	}
	return false
}

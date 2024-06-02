package eth

import (
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"google.golang.org/protobuf/proto"
)

// Att defines common functionality for all attestation types.
type Att interface {
	proto.Message
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	Version() int
	Copy() Att
	GetAggregationBits() bitfield.Bitlist
	GetData() *AttestationData
	CommitteeBitsVal() bitfield.Bitfield
	GetSignature() []byte
}

// IndexedAtt defines common functionality for all indexed attestation types.
type IndexedAtt interface {
	proto.Message
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	Version() int
	GetAttestingIndices() []uint64
	GetData() *AttestationData
	GetSignature() []byte
}

// SignedAggregateAttAndProof defines common functionality for all signed aggregate attestation types.
type SignedAggregateAttAndProof interface {
	proto.Message
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	Version() int
	AggregateAttestationAndProof() AggregateAttAndProof
	GetSignature() []byte
}

// AggregateAttAndProof defines common functionality for all aggregate attestation types.
type AggregateAttAndProof interface {
	proto.Message
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	Version() int
	GetAggregatorIndex() primitives.ValidatorIndex
	AggregateVal() Att
	GetSelectionProof() []byte
}

// AttSlashing defines common functionality for all attestation slashing types.
type AttSlashing interface {
	proto.Message
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	Version() int
	FirstAttestation() IndexedAtt
	SecondAttestation() IndexedAtt
}

// Version --
func (a *Attestation) Version() int {
	return version.Phase0
}

// Copy --
func (a *Attestation) Copy() Att {
	return CopyAttestation(a)
}

// CommitteeBitsVal --
func (a *Attestation) CommitteeBitsVal() bitfield.Bitfield {
	return nil
}

// Version --
func (a *PendingAttestation) Version() int {
	return version.Phase0
}

// Copy --
func (a *PendingAttestation) Copy() Att {
	return CopyPendingAttestation(a)
}

// CommitteeBitsVal --
func (a *PendingAttestation) CommitteeBitsVal() bitfield.Bitfield {
	return nil
}

// GetSignature --
func (a *PendingAttestation) GetSignature() []byte {
	return nil
}

// Version --
func (a *AttestationElectra) Version() int {
	return version.Electra
}

// Copy --
func (a *AttestationElectra) Copy() Att {
	return CopyAttestationElectra(a)
}

// CommitteeBitsVal --
func (a *AttestationElectra) CommitteeBitsVal() bitfield.Bitfield {
	return a.CommitteeBits
}

// Version --
func (a *IndexedAttestation) Version() int {
	return version.Phase0
}

// Version --
func (a *IndexedAttestationElectra) Version() int {
	return version.Electra
}

// Version --
func (a *AttesterSlashing) Version() int {
	return version.Phase0
}

// FirstAttestation --
func (a *AttesterSlashing) FirstAttestation() IndexedAtt {
	return a.Attestation_1
}

// SecondAttestation --
func (a *AttesterSlashing) SecondAttestation() IndexedAtt {
	return a.Attestation_2
}

// Version --
func (a *AttesterSlashingElectra) Version() int {
	return version.Electra
}

// FirstAttestation --
func (a *AttesterSlashingElectra) FirstAttestation() IndexedAtt {
	return a.Attestation_1
}

// SecondAttestation --
func (a *AttesterSlashingElectra) SecondAttestation() IndexedAtt {
	return a.Attestation_2
}

// Version --
func (a *AggregateAttestationAndProof) Version() int {
	return version.Phase0
}

// AggregateVal --
func (a *AggregateAttestationAndProof) AggregateVal() Att {
	return a.Aggregate
}

// Version --
func (a *AggregateAttestationAndProofElectra) Version() int {
	return version.Electra
}

// AggregateVal --
func (a *AggregateAttestationAndProofElectra) AggregateVal() Att {
	return a.Aggregate
}

// Version --
func (a *SignedAggregateAttestationAndProof) Version() int {
	return version.Phase0
}

// AggregateAttestationAndProof --
func (a *SignedAggregateAttestationAndProof) AggregateAttestationAndProof() AggregateAttAndProof {
	return a.Message
}

// Version --
func (a *SignedAggregateAttestationAndProofElectra) Version() int {
	return version.Electra
}

// AggregateAttestationAndProof --
func (a *SignedAggregateAttestationAndProofElectra) AggregateAttestationAndProof() AggregateAttAndProof {
	return a.Message
}

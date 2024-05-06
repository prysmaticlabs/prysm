package eth

import (
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

type Att interface {
	proto.Message
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	Version() int
	Copy() Att
	GetAggregationBits() bitfield.Bitlist
	GetData() *AttestationData
	GetCommitteeBitsVal() bitfield.Bitfield
	GetSignature() []byte
}

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

type SignedAggregateAttAndProof interface {
	proto.Message
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	Version() int
	GetAggregateAttestationAndProof() AggregateAttAndProof
	GetSignature() []byte
}

type AggregateAttAndProof interface {
	proto.Message
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	Version() int
	GetAggregatorIndex() primitives.ValidatorIndex
	GetAggregateVal() Att
	GetSelectionProof() []byte
}

type AttSlashing interface {
	proto.Message
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	Version() int
	GetFirstAttestation() IndexedAtt
	GetSecondAttestation() IndexedAtt
}

func (a *Attestation) Version() int {
	return version.Phase0
}

func (a *Attestation) Copy() Att {
	return CopyAttestation(a)
}

func (a *Attestation) GetCommitteeBitsVal() bitfield.Bitfield {
	return nil
}

func (a *PendingAttestation) Version() int {
	return version.Phase0
}

func (a *PendingAttestation) Copy() Att {
	return CopyPendingAttestation(a)
}

func (a *PendingAttestation) GetCommitteeBitsVal() bitfield.Bitfield {
	return nil
}

func (a *PendingAttestation) GetSignature() []byte {
	return nil
}

func (a *AttestationElectra) Version() int {
	return version.Electra
}

func (a *AttestationElectra) Copy() Att {
	return CopyAttestationElectra(a)
}

func (a *AttestationElectra) GetCommitteeBitsVal() bitfield.Bitfield {
	return a.CommitteeBits
}

func (a *IndexedAttestation) Version() int {
	return version.Phase0
}

func (a *IndexedAttestationElectra) Version() int {
	return version.Electra
}

func (a *AttesterSlashing) Version() int {
	return version.Phase0
}

func (a *AttesterSlashing) GetFirstAttestation() IndexedAtt {
	return a.Attestation_1
}

func (a *AttesterSlashing) GetSecondAttestation() IndexedAtt {
	return a.Attestation_2
}

func (a *AttesterSlashingElectra) Version() int {
	return version.Electra
}

func (a *AttesterSlashingElectra) GetFirstAttestation() IndexedAtt {
	return a.Attestation_1
}

func (a *AttesterSlashingElectra) GetSecondAttestation() IndexedAtt {
	return a.Attestation_2
}

func (a *AggregateAttestationAndProof) Version() int {
	return version.Phase0
}

func (a *AggregateAttestationAndProof) GetAggregateVal() Att {
	return a.Aggregate
}

func (a *AggregateAttestationAndProofElectra) Version() int {
	return version.Electra
}

func (a *AggregateAttestationAndProofElectra) GetAggregateVal() Att {
	return a.Aggregate
}

func (a *SignedAggregateAttestationAndProof) Version() int {
	return version.Phase0
}

func (a *SignedAggregateAttestationAndProof) GetAggregateAttestationAndProof() AggregateAttAndProof {
	return a.Message
}

func (a *SignedAggregateAttestationAndProofElectra) Version() int {
	return version.Electra
}

func (a *SignedAggregateAttestationAndProofElectra) GetAggregateAttestationAndProof() AggregateAttAndProof {
	return a.Message
}

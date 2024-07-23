package eth

import (
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
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
	Clone() Att
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

// Copy --
func (cp *Checkpoint) Copy() *Checkpoint {
	if cp == nil {
		return nil
	}
	return &Checkpoint{
		Epoch: cp.Epoch,
		Root:  bytesutil.SafeCopyBytes(cp.Root),
	}
}

// Copy --
func (attData *AttestationData) Copy() *AttestationData {
	if attData == nil {
		return nil
	}
	return &AttestationData{
		Slot:            attData.Slot,
		CommitteeIndex:  attData.CommitteeIndex,
		BeaconBlockRoot: bytesutil.SafeCopyBytes(attData.BeaconBlockRoot),
		Source:          attData.Source.Copy(),
		Target:          attData.Target.Copy(),
	}
}

// Version --
func (a *Attestation) Version() int {
	return version.Phase0
}

// Clone --
func (a *Attestation) Clone() Att {
	return a.Copy()
}

// Copy --
func (att *Attestation) Copy() *Attestation {
	if att == nil {
		return nil
	}
	return &Attestation{
		AggregationBits: bytesutil.SafeCopyBytes(att.AggregationBits),
		Data:            att.Data.Copy(),
		Signature:       bytesutil.SafeCopyBytes(att.Signature),
	}
}

// CommitteeBitsVal --
func (a *Attestation) CommitteeBitsVal() bitfield.Bitfield {
	cb := primitives.NewAttestationCommitteeBits()
	cb.SetBitAt(uint64(a.Data.CommitteeIndex), true)
	return cb
}

// Version --
func (a *PendingAttestation) Version() int {
	return version.Phase0
}

// Clone --
func (a *PendingAttestation) Clone() Att {
	return a.Copy()
}

// Copy --
func (a *PendingAttestation) Copy() *PendingAttestation {
	if a == nil {
		return nil
	}
	return &PendingAttestation{
		AggregationBits: bytesutil.SafeCopyBytes(a.AggregationBits),
		Data:            a.Data.Copy(),
		InclusionDelay:  a.InclusionDelay,
		ProposerIndex:   a.ProposerIndex,
	}
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

// Clone --
func (a *AttestationElectra) Clone() Att {
	return a.Copy()
}

// Copy --
func (att *AttestationElectra) Copy() *AttestationElectra {
	if att == nil {
		return nil
	}
	return &AttestationElectra{
		AggregationBits: bytesutil.SafeCopyBytes(att.AggregationBits),
		CommitteeBits:   bytesutil.SafeCopyBytes(att.CommitteeBits),
		Data:            att.Data.Copy(),
		Signature:       bytesutil.SafeCopyBytes(att.Signature),
	}
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

// Copy --
func (indexedAtt *IndexedAttestation) Copy() *IndexedAttestation {
	var indices []uint64
	if indexedAtt == nil {
		return nil
	} else if indexedAtt.AttestingIndices != nil {
		indices = make([]uint64, len(indexedAtt.AttestingIndices))
		copy(indices, indexedAtt.AttestingIndices)
	}
	return &IndexedAttestation{
		AttestingIndices: indices,
		Data:             indexedAtt.Data.Copy(),
		Signature:        bytesutil.SafeCopyBytes(indexedAtt.Signature),
	}
}

// Copy --
func (indexedAtt *IndexedAttestationElectra) Copy() *IndexedAttestationElectra {
	var indices []uint64
	if indexedAtt == nil {
		return nil
	} else if indexedAtt.AttestingIndices != nil {
		indices = make([]uint64, len(indexedAtt.AttestingIndices))
		copy(indices, indexedAtt.AttestingIndices)
	}
	return &IndexedAttestationElectra{
		AttestingIndices: indices,
		Data:             indexedAtt.Data.Copy(),
		Signature:        bytesutil.SafeCopyBytes(indexedAtt.Signature),
	}
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

func (a *AttesterSlashing) Copy() *AttesterSlashing {
	if a == nil {
		return nil
	}
	return &AttesterSlashing{
		Attestation_1: a.Attestation_1.Copy(),
		Attestation_2: a.Attestation_2.Copy(),
	}
}

// Copy --
func (a *AttesterSlashingElectra) Copy() *AttesterSlashingElectra {
	if a == nil {
		return nil
	}
	return &AttesterSlashingElectra{
		Attestation_1: a.Attestation_1.Copy(),
		Attestation_2: a.Attestation_2.Copy(),
	}
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

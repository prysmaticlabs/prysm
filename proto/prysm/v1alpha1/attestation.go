package eth

import (
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"google.golang.org/protobuf/proto"
)

// TODO: it would be nicer to declare this inside consensus-types, but this will result in a circular dependency
// (because the interface method returns another interface, the implementation also returns an interface)
// TODO: remove unused methods
type IndexedAtt interface {
	proto.Message
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	Version() int
	GetAttestingIndices() []uint64
	SetAttestingIndices([]uint64)
	GetData() *AttestationData
	SetData(*AttestationData)
	GetSignature() []byte
	SetSignature(sig []byte)
}

func (a *Attestation) Version() int {
	return version.Phase0
}

func (a *Attestation) GetCommitteeBitsVal() bitfield.Bitfield {
	return nil
}

func (a *PendingAttestation) Version() int {
	return version.Phase0
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

func (a *AttestationElectra) GetCommitteeBitsVal() bitfield.Bitfield {
	return a.CommitteeBits
}

func (a *IndexedAttestation) Version() int {
	return version.Phase0
}

func (a *IndexedAttestation) SetAttestingIndices(indices []uint64) {
	a.AttestingIndices = indices
}

func (a *IndexedAttestation) SetData(data *AttestationData) {
	a.Data = data
}

func (a *IndexedAttestation) SetSignature(sig []byte) {
	a.Signature = sig
}

func (a *IndexedAttestationElectra) Version() int {
	return version.Electra
}

func (a *IndexedAttestationElectra) SetAttestingIndices(indices []uint64) {
	a.AttestingIndices = indices
}

func (a *IndexedAttestationElectra) SetData(data *AttestationData) {
	a.Data = data
}

func (a *IndexedAttestationElectra) SetSignature(sig []byte) {
	a.Signature = sig
}

func (a *AttesterSlashing) Version() int {
	return version.Phase0
}

func (a *AttesterSlashing) GetFirstAttestation() IndexedAtt {
	return a.Attestation_1
}
func (a *AttesterSlashing) SetFirstAttestation(att IndexedAtt) {
	at, ok := att.(*IndexedAttestation)
	//TODO: should this error?
	if ok {
		a.Attestation_1 = at
	}
}
func (a *AttesterSlashing) GetSecondAttestation() IndexedAtt {
	return a.Attestation_2
}
func (a *AttesterSlashing) SetSecondAttestation(att IndexedAtt) {
	at, ok := att.(*IndexedAttestation)
	// TODO: should this error?
	if ok {
		a.Attestation_2 = at
	}
}

func (a *AttesterSlashingElectra) Version() int {
	return version.Electra
}

func (a *AttesterSlashingElectra) GetFirstAttestation() IndexedAtt {
	return a.Attestation_1
}

func (a *AttesterSlashingElectra) SetFirstAttestation(att IndexedAtt) {
	at, ok := att.(*IndexedAttestationElectra)
	// TODO: should this error?
	if ok {
		a.Attestation_1 = at
	}
}

func (a *AttesterSlashingElectra) GetSecondAttestation() IndexedAtt {
	return a.Attestation_2
}

func (a *AttesterSlashingElectra) SetSecondAttestation(att IndexedAtt) {
	at, ok := att.(*IndexedAttestationElectra)
	//TODO: should this error?
	if ok {
		a.Attestation_2 = at
	}
}

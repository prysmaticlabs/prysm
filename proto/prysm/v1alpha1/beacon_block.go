package eth

import (
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
)

//TODO: consider replacing this entirely... indexattestation doesn't need to be a protobuf

func (a *AttesterSlashing) GetAttestationOne() interfaces.IndexedAttestation {
	return a.Attestation_1
}
func (a *AttesterSlashing) SetAttestationOne(att interfaces.IndexedAttestation) {
	at, ok := att.(*IndexedAttestation)
	//TODO: should this error?
	if ok {
		a.Attestation_1 = at
	}
}
func (a *AttesterSlashing) GetAttestationTwo() interfaces.IndexedAttestation {
	return a.Attestation_2
}
func (a *AttesterSlashing) SetAttestationTwo(att interfaces.IndexedAttestation) {
	at, ok := att.(*IndexedAttestation)
	//TODO: should this error?
	if ok {
		a.Attestation_2 = at
	}
}

func (a *IndexedAttestation) GetAttestingIndicesVal() []uint64 {
	return a.AttestingIndices
}
func (a *IndexedAttestation) SetAttestingIndicesVal(indices []uint64) {
	a.AttestingIndices = indices
}

func (a *IndexedAttestation) SetData(data *AttestationData) {
	a.Data = data
}

func (a *IndexedAttestation) SetSignature(sig []byte) {
	a.Signature = sig
}

func (a *AttesterSlashingElectra) GetAttestationOne() interfaces.IndexedAttestation {
	return a.Attestation_1
}
func (a *AttesterSlashingElectra) SetAttestationOne(att interfaces.IndexedAttestation) {
	at, ok := att.(*IndexedAttestationElectra)
	//TODO: should this error?
	if ok {
		a.Attestation_1 = at
	}
}
func (a *AttesterSlashingElectra) GetAttestationTwo() interfaces.IndexedAttestation {
	return a.Attestation_2
}
func (a *AttesterSlashingElectra) SetAttestationTwo(att interfaces.IndexedAttestation) {
	at, ok := att.(*IndexedAttestationElectra)
	//TODO: should this error?
	if ok {
		a.Attestation_2 = at
	}
}

func (a *IndexedAttestationElectra) GetAttestingIndicesVal() []uint64 {
	return a.AttestingIndices
}
func (a *IndexedAttestationElectra) SetAttestingIndicesVal(indices []uint64) {
	a.AttestingIndices = indices
}

func (a *IndexedAttestationElectra) SetData(data *AttestationData) {
	a.Data = data
}

func (a *IndexedAttestationElectra) SetSignature(sig []byte) {
	a.Signature = sig
}

package eth

import (
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

func (a *Attestation) Version() int {
	return version.Phase0
}

func (a *Attestation) GetCommitteeBitsVal() bitfield.Bitfield {
	return nil
}

func (a *Attestation) SetAggregationBits(bits bitfield.Bitlist) {
	a.AggregationBits = bits
}

func (a *Attestation) SetData(data *AttestationData) {
	a.Data = data
}

func (a *Attestation) SetCommitteeBitsVal(bits bitfield.Bitfield) {
	return
}

func (a *Attestation) SetSignature(sig []byte) {
	a.Signature = sig
}

func (a *PendingAttestation) Version() int {
	return version.Phase0
}

func (a *PendingAttestation) GetCommitteeBitsVal() bitfield.Bitfield {
	return nil
}

func (a *PendingAttestation) SetAggregationBits(bits bitfield.Bitlist) {
	a.AggregationBits = bits
}

func (a *PendingAttestation) SetData(data *AttestationData) {
	a.Data = data
}

func (a *PendingAttestation) SetCommitteeBitsVal(bits bitfield.Bitfield) {
	return
}

func (a *PendingAttestation) SetSignature(sig []byte) {
	return
}

func (a *PendingAttestation) GetSignature() []byte {
	return nil
}

func (a *AttestationElectra) Version() int {
	return version.Electra
}

func (a *AttestationElectra) SetAggregationBits(bits bitfield.Bitlist) {
	a.AggregationBits = bits
}

func (a *AttestationElectra) SetData(data *AttestationData) {
	a.Data = data
}

func (a *AttestationElectra) GetCommitteeBitsVal() bitfield.Bitfield {
	return a.CommitteeBits
}

func (a *AttestationElectra) SetCommitteeBitsVal(bits bitfield.Bitfield) {
	//TODO: process this based on mainnet vs minimal spec
	//a.CommitteeBits = bits
}

func (a *AttestationElectra) SetSignature(sig []byte) {
	a.Signature = sig
}

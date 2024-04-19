package eth

import (
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

func (a *Attestation) Version() int {
	return version.Phase0
}

func (a *Attestation) GetCommitteeBits() bitfield.Bitlist {
	return nil
}

func (a *Attestation) SetAggregationBits(bits bitfield.Bitlist) {
	a.AggregationBits = bits
}

func (a *Attestation) SetData(data *AttestationData) {
	a.Data = data
}

func (a *Attestation) SetCommitteeBits(bits bitfield.Bitlist) {
	return
}

func (a *Attestation) SetSignature(sig []byte) {
	a.Signature = sig
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

func (a *AttestationElectra) SetCommitteeBits(bits bitfield.Bitlist) {
	a.CommitteeBits = bits
}

func (a *AttestationElectra) SetSignature(sig []byte) {
	a.Signature = sig
}

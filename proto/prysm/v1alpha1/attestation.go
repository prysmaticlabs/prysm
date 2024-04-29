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

func (a *PendingAttestation) Version() int {
	return version.Phase0
}

func (a *PendingAttestation) GetCommitteeBits() bitfield.Bitlist {
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

func (a *AttesterSlashingElectra) Version() int {
	return version.Electra
}

func (a *AttesterSlashingElectra) GetFirstAttestation() IndexedAtt {
	return a.Attestation_1
}

func (a *AttesterSlashingElectra) GetSecondAttestation() IndexedAtt {
	return a.Attestation_2
}

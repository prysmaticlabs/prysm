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

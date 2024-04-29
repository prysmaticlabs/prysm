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

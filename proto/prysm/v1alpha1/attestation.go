package eth

import (
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"google.golang.org/protobuf/proto"
)

// TODO: it would be nicer to declare this inside consensus-types, but this will result in a circular dependency
// (because the interface method returns another interface, the implementation also returns an interface)
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

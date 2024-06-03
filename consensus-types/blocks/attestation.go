package blocks

import (
	"strconv"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// TODO: Doc

type AttestationId struct {
	version int
	digest  [32]byte
}

func (id AttestationId) String() string {
	return strconv.Itoa(id.version) + string(id.digest[:])
}

type ROAttestation struct {
	ethpb.Att
	id     AttestationId
	dataId AttestationId
}

func NewROAttestation(att ethpb.Att) (ROAttestation, error) {
	if err := helpers.ValidateNilAttestation(att); err != nil {
		return ROAttestation{}, err
	}

	attRoot, err := att.HashTreeRoot()
	if err != nil {
		return ROAttestation{}, err
	}
	dataRoot, err := att.GetData().HashTreeRoot()
	if err != nil {
		return ROAttestation{}, err
	}

	return ROAttestation{
		Att:    att,
		id:     AttestationId{version: att.Version(), digest: attRoot},
		dataId: AttestationId{version: att.Version(), digest: dataRoot},
	}, nil
}

func (a ROAttestation) Id() AttestationId {
	return a.id
}

func (a ROAttestation) DataId() AttestationId {
	return a.dataId
}

func (a ROAttestation) Copy() ROAttestation {
	return ROAttestation{
		Att:    a.Att.Copy(),
		id:     a.id,
		dataId: a.dataId,
	}
}

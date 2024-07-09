package attestation

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

// IdSource represents the part of attestation that will be used to generate the Id.
type IdSource uint8

const (
	// Full generates the Id from the whole attestation.
	Full IdSource = iota
	// Data generates the Id from the tuple (slot, committee index, beacon block root, source, target).
	Data
)

// Id represents an attestation ID. Its uniqueness depends on the IdSource provided when constructing the Id.
type Id [33]byte

// NewId --
func NewId(att ethpb.Att, source IdSource) (Id, error) {
	if err := helpers.ValidateNilAttestation(att); err != nil {
		return Id{}, err
	}
	if att.Version() < 0 || att.Version() > 255 {
		return Id{}, errors.New("attestation version must be between 0 and 255")
	}

	var id Id
	id[0] = byte(att.Version())

	switch source {
	case Full:
		h, err := att.HashTreeRoot()
		if err != nil {
			return Id{}, err
		}
		copy(id[1:], h[:])
		return id, nil
	case Data:
		data := att.GetData()
		if att.Version() >= version.Electra {
			committeeIndices := att.CommitteeBitsVal().BitIndices()
			if len(committeeIndices) != 1 {
				return Id{}, fmt.Errorf("%d committee bits are set instead of 1", len(committeeIndices))
			}
			dataCopy := ethpb.CopyAttestationData(att.GetData())
			dataCopy.CommitteeIndex = primitives.CommitteeIndex(committeeIndices[0])
			data = dataCopy
		}
		h, err := data.HashTreeRoot()
		if err != nil {
			return Id{}, err
		}
		copy(id[1:], h[:])
		return id, nil
	default:
		return Id{}, errors.New("invalid source requested")
	}
}

// String --
func (id Id) String() string {
	return string(id[:])
}

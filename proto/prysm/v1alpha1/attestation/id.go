package attestation

import (
	"fmt"
	"strconv"

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
type Id struct {
	version int
	hash    [32]byte
}

// NewId --
func NewId(att ethpb.Att, source IdSource) (Id, error) {
	if err := helpers.ValidateNilAttestation(att); err != nil {
		return Id{}, err
	}

	switch source {
	case Full:
		h, err := att.HashTreeRoot()
		if err != nil {
			return Id{}, err
		}
		return Id{version: att.Version(), hash: h}, nil
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
		return Id{version: att.Version(), hash: h}, nil
	default:
		return Id{}, errors.New("invalid source requested")
	}
}

// String --
func (id Id) String() string {
	return strconv.Itoa(id.version) + string(id.hash[:])
}

package attestation

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

type DigestSource uint8

const (
	Full DigestSource = iota
	Data
)

type Id struct {
	version int
	digest  [32]byte
}

func NewId(att ethpb.Att, digest DigestSource) (Id, error) {
	if err := helpers.ValidateNilAttestation(att); err != nil {
		return Id{}, err
	}

	switch digest {
	case Full:
		d, err := att.HashTreeRoot()
		if err != nil {
			return Id{}, err
		}
		return Id{version: att.Version(), digest: d}, nil
	case Data:
		data := att.GetData()
		if att.Version() >= version.Electra {
			cb := att.CommitteeBitsVal().BitIndices()
			if len(cb) == 0 {
				return Id{}, errors.New("no committee bits set")
			}
			dataCopy := ethpb.CopyAttestationData(att.GetData())
			dataCopy.CommitteeIndex = primitives.CommitteeIndex(cb[0])
			data = dataCopy
		}
		d, err := data.HashTreeRoot()
		if err != nil {
			return Id{}, err
		}
		return Id{version: att.Version(), digest: d}, nil
	default:
		return Id{}, errors.New("invalid digest requested")
	}
}

func (id Id) String() string {
	return strconv.Itoa(id.version) + string(id.digest[:])
}

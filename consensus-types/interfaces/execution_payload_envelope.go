package interfaces

import (
	"github.com/ethereum/go-ethereum/common"
	field_params "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

type ROSignedExecutionPayloadEnvelope interface {
	Envelope() (ROExecutionPayloadEnvelope, error)
	Signature() ([field_params.BLSSignatureLength]byte, error)
	IsNil() bool
}

type ROExecutionPayloadEnvelope interface {
	Execution() (ExecutionData, error)
	BuilderIndex() (primitives.ValidatorIndex, error)
	BeaconBlockRoot() ([field_params.RootLength]byte, error)
	BlobKzgCommitments() ([][]byte, error)
	BlobKzgCommitmentsRoot() ([field_params.RootLength]byte, error)
	VersionedHashes() ([]common.Hash, error)
	PayloadWithheld() (bool, error)
	StateRoot() ([field_params.RootLength]byte, error)
	IsBlinded() bool
	IsNil() bool
}

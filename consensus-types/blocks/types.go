package blocks

import (
	"fmt"

	"github.com/pkg/errors"
	field_params "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	engine "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

var (
	_ = interfaces.SignedBeaconBlock(&SignedBeaconBlock{})
	_ = interfaces.BeaconBlock(&BeaconBlock{})
	_ = interfaces.BeaconBlockBody(&BeaconBlockBody{})
)

const (
	incorrectBlockVersion = "incorrect beacon block version"
	incorrectBodyVersion  = "incorrect beacon block body version"
)

var (
	// ErrUnsupportedGetter is returned when a getter access is not supported for a specific beacon block version.
	ErrUnsupportedGetter = errors.New("unsupported getter")
	// ErrUnsupportedVersion for beacon block methods.
	ErrUnsupportedVersion = errors.New("unsupported beacon block version")
	// ErrNilObjectWrapped is returned in a constructor when the underlying object is nil.
	ErrNilObjectWrapped      = errors.New("attempted to wrap nil object")
	errNilBlock              = errors.New("received nil beacon block")
	errNilBlockBody          = errors.New("received nil beacon block body")
	errIncorrectBlockVersion = errors.New(incorrectBlockVersion)
	errIncorrectBodyVersion  = errors.New(incorrectBodyVersion)
)

// BeaconBlockBody is the main beacon block body structure. It can represent any block type.
type BeaconBlockBody struct {
	syncAggregate          *eth.SyncAggregate
	executionPayloadHeader *engine.ExecutionPayloadHeader
	executionPayload       *engine.ExecutionPayload
	eth1Data               *eth.Eth1Data
	attesterSlashings      []*eth.AttesterSlashing
	proposerSlashings      []*eth.ProposerSlashing
	attestations           []*eth.Attestation
	deposits               []*eth.Deposit
	voluntaryExits         []*eth.SignedVoluntaryExit
	version                int
	randaoReveal           [field_params.BLSSignatureLength]byte
	graffiti               [field_params.RootLength]byte
	isBlinded              bool
}

// BeaconBlock is the main beacon block structure. It can represent any block type.
type BeaconBlock struct {
	body          *BeaconBlockBody
	version       int
	slot          types.Slot
	proposerIndex types.ValidatorIndex
	parentRoot    [field_params.RootLength]byte
	stateRoot     [field_params.RootLength]byte
}

// SignedBeaconBlock is the main signed beacon block structure. It can represent any block type.
type SignedBeaconBlock struct {
	block     *BeaconBlock
	version   int
	signature [field_params.BLSSignatureLength]byte
}

func errNotSupported(funcName string, ver int) error {
	return errors.Wrap(ErrUnsupportedGetter, fmt.Sprintf("%s is not supported for %s", funcName, version.String(ver)))
}

package blocks

import (
	"fmt"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	engine "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

// ErrUnsupportedGetter is returned when a getter access is not supported for a specific beacon block version.
var ErrUnsupportedGetter = errors.New("unsupported getter")

const incorrectBlockVersion = "incorrect beacon block version"
const incorrectBodyVersion = "incorrect beacon block body version"

var errNilBlock = errors.New("received nil beacon block")
var errNilBody = errors.New("received nil beacon block body")
var errIncorrectBlockVersion = errors.New(incorrectBlockVersion)
var errIncorrectBodyVersion = errors.New(incorrectBodyVersion)
var errCloningFailed = errors.New("cloning proto message failed")
var errAssertionFailed = errors.New("type assertion failed")

// BeaconBlockBody is the main beacon block body structure. It can represent any block type.
type BeaconBlockBody struct {
	version                int
	randaoReveal           []byte
	eth1Data               *eth.Eth1Data
	graffiti               []byte
	proposerSlashings      []*eth.ProposerSlashing
	attesterSlashings      []*eth.AttesterSlashing
	attestations           []*eth.Attestation
	deposits               []*eth.Deposit
	voluntaryExits         []*eth.SignedVoluntaryExit
	syncAggregate          *eth.SyncAggregate
	executionPayload       *engine.ExecutionPayload
	executionPayloadHeader *eth.ExecutionPayloadHeader
}

// BeaconBlock is the main beacon block structure. It can represent any block type.
type BeaconBlock struct {
	version       int
	slot          types.Slot
	proposerIndex types.ValidatorIndex
	parentRoot    []byte
	stateRoot     []byte
	body          *BeaconBlockBody
}

// SignedBeaconBlock is the main signed beacon block structure. It can represent any block type.
type SignedBeaconBlock struct {
	version   int
	block     *BeaconBlock
	signature []byte
}

func errNotSupported(funcName string, ver int) error {
	return errors.Wrap(ErrUnsupportedGetter, fmt.Sprintf("%s is not supported for %s", funcName, version.String(ver)))
}

package blocks

import (
	"fmt"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	engine "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

const incorrectBlockVersion = "incorrect beacon block version"
const incorrectBodyVersion = "incorrect beacon block body version"

var errNilBlock = errors.New("received nil beacon block")
var errNilBody = errors.New("received nil beacon block body")
var errIncorrectBlockVersion = errors.New(incorrectBlockVersion)
var errIncorrectBodyVersion = errors.New(incorrectBodyVersion)
var errCloningFailed = errors.New("cloning proto message failed")

// BeaconBlockBody is the main beacon block body structure. It can represent any block type.
type BeaconBlockBody struct {
	version           int
	blinded           bool
	randaoReveal      []byte
	eth1Data          *eth.Eth1Data
	graffiti          []byte
	proposerSlashings []*eth.ProposerSlashing
	attesterSlashings []*eth.AttesterSlashing
	attestations      []*eth.Attestation
	deposits          []*eth.Deposit
	voluntaryExits    []*eth.SignedVoluntaryExit
	syncAggregate     *eth.SyncAggregate
	// TODO: Why these two are from different packages?
	executionPayload       *engine.ExecutionPayload
	executionPayloadHeader *eth.ExecutionPayloadHeader
}

// BeaconBlock is the main beacon block structure. It can represent any block type.
type BeaconBlock struct {
	version       int
	blinded       bool
	slot          types.Slot
	proposerIndex types.ValidatorIndex
	parentRoot    []byte
	stateRoot     []byte
	body          *BeaconBlockBody
}

// SignedBeaconBlock is the main signed beacon block structure. It can represent any block type.
type SignedBeaconBlock struct {
	version   int
	blinded   bool
	block     *BeaconBlock
	signature []byte
}

func errNotSupported(funcName string, ver int) error {
	return fmt.Errorf("%s is not supported for %s", funcName, version.String(ver))
}

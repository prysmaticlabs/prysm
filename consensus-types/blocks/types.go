package blocks

import (
	"fmt"

	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	engine "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

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

type BeaconBlock struct {
	version       int
	blinded       bool
	slot          types.Slot
	proposerIndex types.ValidatorIndex
	parentRoot    []byte
	stateRoot     []byte
	body          *BeaconBlockBody
}

type SignedBeaconBlock struct {
	version   int
	blinded   bool
	block     *BeaconBlock
	signature []byte
}

func errNotSupported(funcName string, ver int) error {
	return fmt.Errorf("%s is not supported for %s", funcName, version.String(ver))
}

package blocks

import (
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	engine "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

type BeaconBlockBody struct {
	version           int
	blinded           bool
	RandaoReveal      []byte
	Eth1Data          *eth.Eth1Data
	Graffiti          []byte
	ProposerSlashings []*eth.ProposerSlashing
	AttesterSlashings []*eth.AttesterSlashing
	Attestations      []*eth.Attestation
	Deposits          []*eth.Deposit
	VoluntaryExits    []*eth.SignedVoluntaryExit
	SyncAggregate     *eth.SyncAggregate
	// TODO: Why these two are from different packages?
	ExecutionPayload       *engine.ExecutionPayload
	ExecutionPayloadHeader *eth.ExecutionPayloadHeader
}

type BeaconBlock struct {
	version       int
	blinded       bool
	Slot          types.Slot
	ProposerIndex types.ValidatorIndex
	ParentRoot    []byte
	StateRoot     []byte
	Body          *BeaconBlockBody
}

type SingedBeaconBlock struct {
	version   int
	blinded   bool
	Block     *BeaconBlock
	Signature []byte
}

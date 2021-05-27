// Package block contains types for block-specific events fired
// during the runtime of a beacon node.
package block

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	vanTypes "github.com/prysmaticlabs/prysm/shared/params"
)

const (
	// ReceivedBlock is sent after a block has been received by the beacon node via p2p or RPC.
	ReceivedBlock = iota + 1
	// UnconfirmedBlock is sent after a block has been processed by the beacon node but not confirmed by orchestrator
	UnConfirmedBlock
	// ConfirmedBlock is sent after a block has been confirmed by orchestrator client
	ConfirmedBlock
)

// ReceivedBlockData is the data sent with ReceivedBlock events.
type ReceivedBlockData struct {
	SignedBlock *ethpb.SignedBeaconBlock
}

// UnConfirmedBlockData is the data sent to orchestrator
type UnConfirmedBlockData struct {
	Block *ethpb.BeaconBlock
}

// ConfirmedData is the data which is sent after getting confirmation from orchestrator
type ConfirmedData struct {
	Slot          types.Slot
	BlockRootHash [32]byte
	Status        vanTypes.Status
}

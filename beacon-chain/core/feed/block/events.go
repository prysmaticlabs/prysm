package block

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

const (
	// ReceivedBlock is sent after a block has been received by the beacon node via p2p or RPC.
	ReceivedBlock = iota + 1
)

// ReceivedBlockData is the data sent with ReceivedBlock events.
type ReceivedBlockData struct {
	SignedBlock *ethpb.SignedBeaconBlock
}

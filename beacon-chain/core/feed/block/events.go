package block

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

const (
	// BlockReceived is sent after a block has been received by the beacon node via p2p or RPC.
	BlockReceived = iota + 1
)

// BlockReceivedData is the data sent with BlockProcessed events.
type BlockReceivedData struct {
	SignedBlock *ethpb.SignedBeaconBlock
}

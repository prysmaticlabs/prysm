// Package block contains types for block-specific events fired
// during the runtime of a beacon node.
package block

import (
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
)

const (
	// ReceivedBlock is sent after a block has been received by the beacon node via p2p or RPC.
	ReceivedBlock = iota + 1
)

// ReceivedBlockData is the data sent with ReceivedBlock events.
type ReceivedBlockData struct {
	SignedBlock  block.SignedBeaconBlock
	IsOptimistic bool
}

package peers

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

// Info provides information about a peer connection.
type Info struct {
	AddrInfo  *peer.AddrInfo
	Direction network.Direction
}

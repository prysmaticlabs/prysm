package p2p

import (
	"context"

	"github.com/ethereum/go-ethereum/sharding/p2p/protocol"
	host "github.com/libp2p/go-libp2p-host"
)

// Node type - a p2p host implementing one or more p2p protocols
type node struct {
	host host.Host
	ping *protocol.PingProtocol
}

func newNode(ctx context.Context, host host.Host) *node {

	return nil
}

func (n *node) Start() {

}

func (n *node) Stop() error {
	return nil
}

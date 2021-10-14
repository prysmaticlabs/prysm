package v2

import "github.com/prysmaticlabs/prysm/beacon-chain/p2p"

func WithP2POptions(opts []p2p.Option) Option {
	return func(node *BeaconNode) error {
		node.p2pOpts = opts
		return nil
	}
}

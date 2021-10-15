package node

import "github.com/prysmaticlabs/prysm/beacon-chain/blockchain"

// Option for beacon node configuration.
type Option func(bn *BeaconNode) error

// WithBlockchainOptions includes functional options for the blockchain service.
func WithBlockchainOptions(opts []blockchain.Option) Option {
	return func(bn *BeaconNode) error {
		bn.blockchainOpts = opts
		return nil
	}
}

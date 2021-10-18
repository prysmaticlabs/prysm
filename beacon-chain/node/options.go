package node

import "github.com/prysmaticlabs/prysm/beacon-chain/blockchain"

// Option for beacon node configuration.
type Option func(bn *BeaconNode) error

// WithBlockchainFlagOptions includes functional options for the blockchain service related to CLI flags.
func WithBlockchainFlagOptions(opts []blockchain.Option) Option {
	return func(bn *BeaconNode) error {
		bn.blockchainFlagOpts = opts
		return nil
	}
}

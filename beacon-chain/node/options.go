package node

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
)

// Option for beacon node configuration.
type Option func(bn *BeaconNode) error

// WithBlockchainFlagOptions includes functional options for the blockchain service related to CLI flags.
func WithBlockchainFlagOptions(opts []blockchain.Option) Option {
	return func(bn *BeaconNode) error {
		bn.serviceFlagOpts.blockchainFlagOpts = opts
		return nil
	}
}

// WithPowchainFlagOptions includes functional options for the powchain service related to CLI flags.
func WithPowchainFlagOptions(opts []powchain.Option) Option {
	return func(bn *BeaconNode) error {
		bn.serviceFlagOpts.powchainFlagOpts = opts
		return nil
	}
}

// WithP2PFlagOptions includes functional options for the p2p service related to CLI flags.
func WithP2PFlagOptions(opts []p2p.Option) Option {
	return func(bn *BeaconNode) error {
		bn.serviceFlagOpts.p2pFlagOpts = opts
		return nil
	}
}

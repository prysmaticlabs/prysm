package accounts

import (
	"context"

	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"google.golang.org/grpc"
)

// NewCLIManager allows for managing validator accounts via CLI commands.
func NewCLIManager(_ context.Context, opts ...Option) (*AccountsCLIManager, error) {
	acc := &AccountsCLIManager{}
	for _, opt := range opts {
		if err := opt(acc); err != nil {
			return nil, err
		}
	}
	return acc, nil
}

// AccountsCLIManager defines a struct capable of performing various validator
// wallet account operations via the command line.
type AccountsCLIManager struct {
	wallet               *wallet.Wallet
	keymanager           keymanager.IKeymanager
	showDepositData      bool
	showPrivateKeys      bool
	listValidatorIndices bool
	dialOpts             []grpc.DialOption
	grpcHeaders          []string
	beaconRPCProvider    string
}

// Option type for configuring the accounts cli manager.
type Option func(acc *AccountsCLIManager) error

// WithWallet provides a wallet to the accounts cli manager.
func WithWallet(wallet *wallet.Wallet) Option {
	return func(acc *AccountsCLIManager) error {
		acc.wallet = wallet
		return nil
	}
}

// WithKeymanager provides a keymanager to the accounts cli manager.
func WithKeymanager(km keymanager.IKeymanager) Option {
	return func(acc *AccountsCLIManager) error {
		acc.keymanager = km
		return nil
	}
}

// WithShowDepositData enables displaying deposit data in the accounts cli manager.
func WithShowDepositData() Option {
	return func(acc *AccountsCLIManager) error {
		acc.showDepositData = true
		return nil
	}
}

// WithShowPrivateKeys enables displaying private keys in the accounts cli manager.
func WithShowPrivateKeys() Option {
	return func(acc *AccountsCLIManager) error {
		acc.showPrivateKeys = true
		return nil
	}
}

// WithListValidatorIndices enables displaying validator indices in the accounts cli manager.
func WithListValidatorIndices() Option {
	return func(acc *AccountsCLIManager) error {
		acc.listValidatorIndices = true
		return nil
	}
}

// WithGRPCDialOpts adds grpc opts needed to connect to beacon nodes in the accounts cli manager.
func WithGRPCDialOpts(opts []grpc.DialOption) Option {
	return func(acc *AccountsCLIManager) error {
		acc.dialOpts = opts
		return nil
	}
}

// WithGRPCHeaders adds grpc headers used when connecting to beacon nodes in the accounts cli manager.
func WithGRPCHeaders(headers []string) Option {
	return func(acc *AccountsCLIManager) error {
		acc.grpcHeaders = headers
		return nil
	}
}

// WithBeaconRPCProvider provides a beacon node endpoint to the accounts cli manager.
func WithBeaconRPCProvider(provider string) Option {
	return func(acc *AccountsCLIManager) error {
		acc.beaconRPCProvider = provider
		return nil
	}
}

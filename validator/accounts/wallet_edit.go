package accounts

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote"
)

// WalletEdit changes a user's on-disk wallet configuration: remote gRPC
// credentials for remote signing, derivation paths for HD wallets, etc.
func (acm *AccountsCLIManager) WalletEdit(ctx context.Context) error {
	encodedCfg, err := remote.MarshalOptionsFile(ctx, acm.keymanagerOpts)
	if err != nil {
		return errors.Wrap(err, "could not marshal config file")
	}
	if err := acm.wallet.WriteKeymanagerConfigToDisk(ctx, encodedCfg); err != nil {
		return errors.Wrap(err, "could not write config to disk")
	}
	return nil
}

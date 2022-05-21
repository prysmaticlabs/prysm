package accounts

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/urfave/cli/v2"
)

func walletWithKeymanager(c *cli.Context) (*wallet.Wallet, keymanager.IKeymanager, error) {
	w, err := wallet.OpenWalletOrElseCli(c, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		return nil, wallet.ErrNoWalletFound
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not open wallet")
	}
	// TODO(#9883) - Remove this when we have a better way to handle this. this is fine.
	// genesis root is not set here which is used for sign function, but fetch keys should be fine.
	km, err := w.InitializeKeymanager(c.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	if err != nil && strings.Contains(err.Error(), keymanager.IncorrectPasswordErrMsg) {
		return nil, nil, errors.New("wrong wallet password entered")
	}
	if err != nil {
		return nil, nil, errors.Wrap(err, accounts.ErrCouldNotInitializeKeymanager)
	}
	return w, km, nil
}

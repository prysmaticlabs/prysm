package accounts

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v3/validator/node"
	"github.com/urfave/cli/v2"
)

func walletWithKeymanager(c *cli.Context, genesisValidatorsRoot []byte) (*wallet.Wallet, keymanager.IKeymanager, error) {
	if c.IsSet(flags.Web3SignerURLFlag.Name) {
		config, err := node.Web3SignerConfig(c)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "could not configure web3signer")
		}
		w := wallet.NewWalletForWeb3Signer()
		config.GenesisValidatorsRoot = genesisValidatorsRoot
		km, err := w.InitializeKeymanager(c.Context, iface.InitKeymanagerConfig{ListenForChanges: false, Web3SignerConfig: config})
		if err != nil {
			return nil, nil, err
		}
		return w, km, nil
	} else {
		w, err := wallet.OpenWalletOrElseCli(c, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
			return nil, wallet.ErrNoWalletFound
		})
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not open wallet")
		}
		km, err := w.InitializeKeymanager(c.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
		if err != nil && strings.Contains(err.Error(), keymanager.IncorrectPasswordErrMsg) {
			return nil, nil, errors.New("wrong wallet password entered")
		}
		if err != nil {
			return nil, nil, errors.Wrap(err, accounts.ErrCouldNotInitializeKeymanager)
		}
		return w, km, nil
	}

}

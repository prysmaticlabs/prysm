package v2

import (
	"strings"

	"github.com/pkg/errors"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/urfave/cli/v2"
)

// SendDeposit transaction.
func SendDeposit(cliCtx *cli.Context) error {
	// Read the wallet from the specified path.
	wallet, err := OpenWallet(cliCtx)
	if errors.Is(err, ErrNoWalletFound) {
		return errors.Wrap(err, "no wallet found at path, create a new wallet with wallet-v2 create")
	} else if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	keymanager, err := wallet.InitializeKeymanager(
		cliCtx,
		true, /* skip mnemonic confirm */
	)
	if err != nil && strings.Contains(err.Error(), "invalid checksum") {
		return errors.New("wrong wallet password entered")
	}
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	switch wallet.KeymanagerKind() {
	case v2keymanager.Derived:
		km, ok := keymanager.(*derived.Keymanager)
		if !ok {
			return errors.New("could not assert keymanager interface to concrete type")
		}
		depositConfig, err := createDepositConfig(cliCtx)
		if err != nil {
			return err
		}
		if err := km.SendDepositTx(depositConfig); err != nil {
			return err
		}
	default:
		return errors.New("only Prysm HD wallets support sending deposits at the moment")
	}
	return nil
}

func createDepositConfig(cliCtx *cli.Context) (*derived.SendDepositConfig, error) {
	return nil, nil
}

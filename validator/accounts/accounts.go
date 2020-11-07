package accounts

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/prompt"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/urfave/cli/v2"
)

// AccountConfig specifies parameters to run to delete, enable, disable accounts.
type AccountConfig struct {
	Wallet     *wallet.Wallet
	Keymanager keymanager.IKeymanager
	PublicKeys [][]byte
}

func SelectPublicKeysFromAccount(cliCtx *cli.Context, pubKeyFlag *cli.StringFlag) (*AccountConfig, error) {
	w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		return nil, wallet.ErrNoWalletFound
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not open wallet")
	}
	keymanager, err := w.InitializeKeymanager(cliCtx.Context, &iface.InitializeKeymanagerConfig{
		SkipMnemonicConfirm: false,
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize keymanager")
	}
	validatingPublicKeys, err := keymanager.FetchValidatingPublicKeys(cliCtx.Context)
	if err != nil {
		return nil, err
	}
	if len(validatingPublicKeys) == 0 {
		return nil, errors.New("wallet is empty")
	}
	// Allow the user to interactively select the accounts or optionally
	// provide them via cli flags as a string of comma-separated, hex strings.
	filteredPubKeys, err := filterPublicKeysFromUserInput(
		cliCtx,
		pubKeyFlag,
		validatingPublicKeys,
		prompt.SelectAccountsDeletePromptText,
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not filter public keys")
	}
	rawPublicKeys := make([][]byte, len(filteredPubKeys))
	for i, pk := range filteredPubKeys {
		pubKeyBytes := pk.Marshal()
		rawPublicKeys[i] = pubKeyBytes
	}
	cfg := &AccountConfig{
		Wallet:     w,
		Keymanager: keymanager,
		PublicKeys: rawPublicKeys,
	}
	return cfg, nil
}

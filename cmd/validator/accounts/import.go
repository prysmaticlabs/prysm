package accounts

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/cmd"
	"github.com/prysmaticlabs/prysm/v3/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/v3/io/prompt"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/userprompt"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v3/validator/client"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/urfave/cli/v2"
)

func accountsImport(c *cli.Context) error {
	w, err := walletImport(c)
	if err != nil {
		return errors.Wrap(err, "could not initialize wallet")
	}
	km, err := w.InitializeKeymanager(c.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	if err != nil {
		return err
	}

	dialOpts := client.ConstructDialOptions(
		c.Int(cmd.GrpcMaxCallRecvMsgSizeFlag.Name),
		c.String(flags.CertFlag.Name),
		c.Uint(flags.GrpcRetriesFlag.Name),
		c.Duration(flags.GrpcRetryDelayFlag.Name),
	)
	grpcHeaders := strings.Split(c.String(flags.GrpcHeadersFlag.Name), ",")

	opts := []accounts.Option{
		accounts.WithWallet(w),
		accounts.WithKeymanager(km),
		accounts.WithGRPCDialOpts(dialOpts),
		accounts.WithBeaconRPCProvider(c.String(flags.BeaconRPCProviderFlag.Name)),
		accounts.WithGRPCHeaders(grpcHeaders),
	}

	opts = append(opts, accounts.WithImportPrivateKeys(c.IsSet(flags.ImportPrivateKeyFileFlag.Name)))
	opts = append(opts, accounts.WithPrivateKeyFile(c.String(flags.ImportPrivateKeyFileFlag.Name)))
	opts = append(opts, accounts.WithReadPasswordFile(c.IsSet(flags.AccountPasswordFileFlag.Name)))
	opts = append(opts, accounts.WithPasswordFilePath(c.String(flags.AccountPasswordFileFlag.Name)))

	keysDir, err := userprompt.InputDirectory(c, userprompt.ImportKeysDirPromptText, flags.KeysDirFlag)
	if err != nil {
		return errors.Wrap(err, "could not parse keys directory")
	}
	opts = append(opts, accounts.WithKeysDir(keysDir))

	acc, err := accounts.NewCLIManager(opts...)
	if err != nil {
		return err
	}
	return acc.Import(c.Context)
}

func walletImport(c *cli.Context) (*wallet.Wallet, error) {
	return wallet.OpenWalletOrElseCli(c, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		walletDir, err := userprompt.InputDirectory(cliCtx, userprompt.WalletDirPromptText, flags.WalletDirFlag)
		if err != nil {
			return nil, err
		}
		exists, err := wallet.Exists(walletDir)
		if err != nil {
			return nil, errors.Wrap(err, wallet.CheckExistsErrMsg)
		}
		if exists {
			isValid, err := wallet.IsValid(walletDir)
			if err != nil {
				return nil, errors.Wrap(err, wallet.CheckValidityErrMsg)
			}
			if !isValid {
				return nil, errors.New(wallet.InvalidWalletErrMsg)
			}
			walletPassword, err := wallet.InputPassword(
				cliCtx,
				flags.WalletPasswordFileFlag,
				wallet.PasswordPromptText,
				false, /* Do not confirm password */
				wallet.ValidateExistingPass,
			)
			if err != nil {
				return nil, err
			}
			return wallet.OpenWallet(cliCtx.Context, &wallet.Config{
				WalletDir:      walletDir,
				WalletPassword: walletPassword,
			})
		}

		wCfg, err := ExtractWalletDirPassword(cliCtx)
		if err != nil {
			return nil, err
		}
		w := wallet.New(&wallet.Config{
			KeymanagerKind: keymanager.Local,
			WalletDir:      wCfg.Dir,
			WalletPassword: wCfg.Password,
		})
		if err = accounts.CreateLocalKeymanagerWallet(cliCtx.Context, w); err != nil {
			return nil, errors.Wrap(err, "could not create keymanager")
		}
		log.WithField("wallet-path", wCfg.Dir).Info(
			"Successfully created new wallet",
		)
		return w, nil
	})
}

// WalletDirPassword holds the directory and password of a wallet.
type WalletDirPassword struct {
	Dir      string
	Password string
}

// ExtractWalletDirPassword prompts the user for wallet directory and password.
func ExtractWalletDirPassword(cliCtx *cli.Context) (WalletDirPassword, error) {
	// Get wallet dir and check that no wallet exists at the location.
	walletDir, err := userprompt.InputDirectory(cliCtx, userprompt.WalletDirPromptText, flags.WalletDirFlag)
	if err != nil {
		return WalletDirPassword{}, err
	}
	walletPassword, err := prompt.InputPassword(
		cliCtx,
		flags.WalletPasswordFileFlag,
		wallet.NewWalletPasswordPromptText,
		wallet.ConfirmPasswordPromptText,
		true, /* Should confirm password */
		prompt.ValidatePasswordInput,
	)
	if err != nil {
		return WalletDirPassword{}, err
	}
	return WalletDirPassword{
		Dir:      walletDir,
		Password: walletPassword,
	}, nil
}

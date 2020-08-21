package v2

import (
	"context"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/urfave/cli/v2"
)

// SendDeposit transaction for user specified accounts via an interactive
// CLI process or via command-line flags.
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
		depositConfig, err := createDepositConfig(cliCtx, km)
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

func createDepositConfig(cliCtx *cli.Context, km *derived.Keymanager) (*derived.SendDepositConfig, error) {
	pubKeys, err := km.FetchValidatingPublicKeys(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch validating public keys")
	}
	// Allow the user to interactively select the accounts to backup or optionally
	// provide them via cli flags as a string of comma-separated, hex strings.
	filteredPubKeys, err := filterPublicKeysFromUserInput(
		cliCtx,
		flags.DepositPublicKeysFlag,
		pubKeys,
		selectAccountsDepositPromptText,
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not filter validating public keys for deposit")
	}

	// Enter the web3provider information.
	web3Provider, err := promptutil.DefaultAndValidatePrompt(
		"Enter the HTTP address of your eth1 endpoint for the Goerli testnet",
		cliCtx.String(flags.HTTPWeb3ProviderFlag.Name),
		func(input string) error {
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	if web3Provider == "" {
		web3Provider = cliCtx.String(flags.HTTPWeb3ProviderFlag.Name)
	}
	depositDelaySeconds := cliCtx.Int(flags.DepositDelaySecondsFlag.Name)
	config := &derived.SendDepositConfig{
		DepositContractAddress: cliCtx.String(flags.DepositContractAddressFlag.Name),
		DepositDelaySeconds:    time.Duration(depositDelaySeconds) * time.Second,
		DepositPublicKeys:      filteredPubKeys,
		Web3Provider:           web3Provider,
	}

	// If the user passes any of the specified flags, we read them and return the
	// config struct directly, bypassing any CLI input.
	hasPrivateKey := cliCtx.IsSet(flags.Eth1PrivateKeyFileFlag.Name)
	hasEth1Keystore := cliCtx.IsSet(flags.Eth1KeystoreUTCPathFlag.Name)
	if hasPrivateKey || hasEth1Keystore {
		if hasPrivateKey {
			fileBytes, err := fileutil.ReadFileAsBytes(cliCtx.String(flags.Eth1PrivateKeyFileFlag.Name))
			if err != nil {
				return nil, err
			}
			config.Eth1PrivateKey = string(fileBytes)
		}
		return config, nil
	}

	usePrivateKeyPrompt := "Inputting an eth1 private key hex string directly"
	useEth1KeystorePrompt := "Using an encrypted eth1 keystore UTC file"
	eth1Prompt := promptui.Select{
		Label: "Select how you wish to sign your eth1 transaction",
		Items: []string{
			usePrivateKeyPrompt,
			useEth1KeystorePrompt,
		},
	}
	_, selection, err := eth1Prompt.Run()
	if err != nil {
		return nil, err
	}
	// If the user wants to proceed by inputting their private key directly, ask for it securely.
	if selection == usePrivateKeyPrompt {
		eth1PrivateKeyString, err := promptutil.PasswordPrompt(
			"Enter the hex string value of your eth1 private key",
			promptutil.NotEmpty,
		)
		if err != nil {
			return nil, err
		}
		config.Eth1PrivateKey = eth1PrivateKeyString
	} else if selection == useEth1KeystorePrompt {
		// Otherwise, ask the user for paths to their keystore UTC file and its password.
		eth1KeystoreUTCFile, err := promptutil.DefaultAndValidatePrompt(
			"Enter the file path for your encrypted, eth1 keystore-utc file",
			cliCtx.String(flags.Eth1KeystoreUTCPathFlag.Name),
			func(input string) error {
				return nil
			},
		)
		if err != nil {
			return nil, err
		}
		eth1KeystorePasswordFile, err := inputWeakPassword(
			cliCtx,
			flags.Eth1KeystorePasswordFileFlag,
			"Enter the file path a .txt file containing your eth1 keystore password",
		)
		if err != nil {
			return nil, err
		}
		config.Eth1KeystoreUTCFile = eth1KeystoreUTCFile
		config.Eth1KeystorePasswordFile = eth1KeystorePasswordFile
	}
	return config, nil
}

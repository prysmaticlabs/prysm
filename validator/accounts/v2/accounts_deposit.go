package v2

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/prompt"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/urfave/cli/v2"
)

// SendDepositCli transaction for user specified accounts via an interactive
// CLI process or via command-line flags.
func SendDepositCli(cliCtx *cli.Context) error {
	w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		return nil, errors.New(
			"no wallet found, nothing to deposit",
		)
	})
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	keymanager, err := w.InitializeKeymanager(
		cliCtx.Context,
		true, /* skip mnemonic confirm */
	)
	if err != nil && strings.Contains(err.Error(), "invalid checksum") {
		return errors.New("wrong wallet password entered")
	}
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	switch w.KeymanagerKind() {
	case v2keymanager.Derived:
		km, ok := keymanager.(*derived.Keymanager)
		if !ok {
			return errors.New("could not assert keymanager interface to concrete type")
		}
		depositConfig, err := createDepositConfig(cliCtx, km)
		if err != nil {
			return errors.Wrap(err, "could not initialize deposit config")
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
	pubKeysBytes, err := km.FetchValidatingPublicKeys(cliCtx.Context)
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch validating public keys")
	}
	pubKeys := make([]bls.PublicKey, len(pubKeysBytes))
	for i, pk := range pubKeysBytes {
		pubKeys[i], err = bls.PublicKeyFromBytes(pk[:])
		if err != nil {
			return nil, errors.Wrap(err, "could not parse BLS public key")
		}
	}
	// Allow the user to interactively select the accounts to deposit or optionally
	// provide them via cli flags as a string of comma-separated, hex strings. If the user has
	// selected to deposit all accounts, we skip this part.
	if !cliCtx.IsSet(flags.DepositAllAccountsFlag.Name) {
		pubKeys, err = filterPublicKeysFromUserInput(
			cliCtx,
			flags.DepositPublicKeysFlag,
			pubKeysBytes,
			prompt.SelectAccountsDepositPromptText,
		)
		if err != nil {
			return nil, errors.Wrap(err, "could not filter validating public keys for deposit")
		}
	}

	web3Provider := cliCtx.String(flags.HTTPWeb3ProviderFlag.Name)
	// Enter the web3provider information.
	if web3Provider == "" {
		web3Provider, err = promptutil.DefaultAndValidatePrompt(
			"Enter the HTTP address of your eth1 endpoint for the Goerli testnet",
			cliCtx.String(flags.HTTPWeb3ProviderFlag.Name),
			func(input string) error {
				return nil
			},
		)
		if err != nil {
			return nil, errors.Wrap(err, "could not validate web3 provider endpoint")
		}
	}
	depositDelaySeconds := cliCtx.Int(flags.DepositDelaySecondsFlag.Name)
	config := &derived.SendDepositConfig{
		DepositContractAddress: cliCtx.String(flags.DepositContractAddressFlag.Name),
		DepositDelaySeconds:    time.Duration(depositDelaySeconds) * time.Second,
		DepositPublicKeys:      pubKeys,
		Web3Provider:           web3Provider,
	}

	if !cliCtx.Bool(flags.SkipDepositConfirmationFlag.Name) {
		confirmDepositPrompt := "You are about to send %d ETH into contract address %s for %d eth2 validator accounts. " +
			"Are you sure you want to do this? Enter the words 'yes I do' to continue"
		gweiPerEth := params.BeaconConfig().GweiPerEth
		ethDepositTotal := uint64(len(pubKeys)) * params.BeaconConfig().MaxEffectiveBalance / gweiPerEth
		if _, err := promptutil.ValidatePrompt(
			os.Stdin,
			fmt.Sprintf(confirmDepositPrompt, ethDepositTotal, config.DepositContractAddress, len(pubKeys)),
			func(input string) error {
				if input != "yes I do" {
					return errors.New("please enter 'yes I do' or exit")
				}
				return nil
			},
		); err != nil {
			return nil, errors.Wrap(err, "could not confirm deposit acknowledgement")
		}
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
			config.Eth1PrivateKey = strings.TrimRight(string(fileBytes), "\r\n")
		} else {
			config.Eth1KeystoreUTCFile = cliCtx.String(flags.Eth1KeystoreUTCPathFlag.Name)
			if cliCtx.IsSet(flags.Eth1KeystorePasswordFileFlag.Name) {
				config.Eth1KeystorePasswordFile = cliCtx.String(flags.Eth1KeystorePasswordFileFlag.Name)
			} else {
				config.Eth1KeystorePasswordFile, err = prompt.InputWeakPassword(
					cliCtx,
					flags.Eth1KeystorePasswordFileFlag,
					"Enter the file path of a text file containing your eth1 keystore password",
				)
				if err != nil {
					return nil, errors.Wrap(err, "could not read eth1 keystore password file path")
				}
			}
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
			return nil, errors.Wrap(err, "could not read eth1 private key string")
		}
		config.Eth1PrivateKey = strings.TrimRight(eth1PrivateKeyString, "\r\n")
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
			return nil, errors.Wrap(err, "could not read eth1 keystore UTC path")
		}
		eth1KeystorePasswordFile, err := prompt.InputWeakPassword(
			cliCtx,
			flags.Eth1KeystorePasswordFileFlag,
			"Enter the file path to a text file containing your eth1 keystore password",
		)
		if err != nil {
			return nil, errors.Wrap(err, "could not read eth1 keystore password file path")
		}
		config.Eth1KeystoreUTCFile = eth1KeystoreUTCFile
		config.Eth1KeystorePasswordFile = eth1KeystorePasswordFile
	}
	return config, nil
}

package accounts

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/grpcutils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/prompt"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/client"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

type performExitCfg struct {
	validatorClient  ethpb.BeaconNodeValidatorClient
	nodeClient       ethpb.NodeClient
	keymanager       keymanager.IKeymanager
	rawPubKeys       [][]byte
	formattedPubKeys []string
}

const exitPassphrase = "Exit my validator"

// ExitAccountsCli performs a voluntary exit on one or more accounts.
func ExitAccountsCli(cliCtx *cli.Context, r io.Reader) error {
	validatingPublicKeys, kManager, err := prepareWallet(cliCtx)
	if err != nil {
		return err
	}

	rawPubKeys, trimmedPubKeys, err := interact(cliCtx, r, validatingPublicKeys)
	if err != nil {
		return err
	}
	// User decided to cancel the voluntary exit.
	if rawPubKeys == nil && trimmedPubKeys == nil {
		return nil
	}

	validatorClient, nodeClient, err := prepareClients(cliCtx)
	if err != nil {
		return err
	}

	cfg := performExitCfg{
		*validatorClient,
		*nodeClient,
		kManager,
		rawPubKeys,
		trimmedPubKeys,
	}
	rawExitedKeys, trimmedExitedKeys, err := performExit(cliCtx, cfg)
	if err != nil {
		return err
	}
	displayExitInfo(rawExitedKeys, trimmedExitedKeys)

	return nil
}

func prepareWallet(cliCtx *cli.Context) (validatingPublicKeys [][48]byte, km keymanager.IKeymanager, err error) {
	w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		return nil, wallet.ErrNoWalletFound
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not open wallet")
	}

	km, err = w.InitializeKeymanager(cliCtx.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	if err != nil {
		return nil, nil, errors.Wrap(err, ErrCouldNotInitializeKeymanager)
	}
	validatingPublicKeys, err = km.FetchValidatingPublicKeys(cliCtx.Context)
	if err != nil {
		return nil, nil, err
	}
	if len(validatingPublicKeys) == 0 {
		return nil, nil, errors.New("wallet is empty, no accounts to perform voluntary exit")
	}

	return validatingPublicKeys, km, nil
}

func interact(
	cliCtx *cli.Context,
	r io.Reader,
	validatingPublicKeys [][48]byte,
) (rawPubKeys [][]byte, formattedPubKeys []string, err error) {
	if !cliCtx.IsSet(flags.ExitAllFlag.Name) {
		// Allow the user to interactively select the accounts to exit or optionally
		// provide them via cli flags as a string of comma-separated, hex strings.
		filteredPubKeys, err := filterPublicKeysFromUserInput(
			cliCtx,
			flags.VoluntaryExitPublicKeysFlag,
			validatingPublicKeys,
			prompt.SelectAccountsVoluntaryExitPromptText,
		)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not filter public keys for voluntary exit")
		}
		rawPubKeys = make([][]byte, len(filteredPubKeys))
		formattedPubKeys = make([]string, len(filteredPubKeys))
		for i, pk := range filteredPubKeys {
			pubKeyBytes := pk.Marshal()
			rawPubKeys[i] = pubKeyBytes
			formattedPubKeys[i] = fmt.Sprintf("%#x", bytesutil.Trunc(pubKeyBytes))
		}
		allAccountStr := strings.Join(formattedPubKeys, ", ")
		if !cliCtx.IsSet(flags.VoluntaryExitPublicKeysFlag.Name) {
			if len(filteredPubKeys) == 1 {
				promptText := "Are you sure you want to perform a voluntary exit on 1 account? (%s) Y/N"
				resp, err := promptutil.ValidatePrompt(
					r, fmt.Sprintf(promptText, au.BrightGreen(formattedPubKeys[0])), promptutil.ValidateYesOrNo,
				)
				if err != nil {
					return nil, nil, err
				}
				if strings.EqualFold(resp, "n") {
					return nil, nil, nil
				}
			} else {
				promptText := "Are you sure you want to perform a voluntary exit on %d accounts? (%s) Y/N"
				if len(filteredPubKeys) == len(validatingPublicKeys) {
					promptText = fmt.Sprintf(
						"Are you sure you want to perform a voluntary exit on all accounts? Y/N (%s)",
						au.BrightGreen(allAccountStr))
				} else {
					promptText = fmt.Sprintf(promptText, len(filteredPubKeys), au.BrightGreen(allAccountStr))
				}
				resp, err := promptutil.ValidatePrompt(r, promptText, promptutil.ValidateYesOrNo)
				if err != nil {
					return nil, nil, err
				}
				if strings.EqualFold(resp, "n") {
					return nil, nil, nil
				}
			}
		}
	} else {
		rawPubKeys = make([][]byte, len(validatingPublicKeys))
		formattedPubKeys = make([]string, len(validatingPublicKeys))
		for i, pk := range validatingPublicKeys {
			rawPubKeys[i] = pk[:]
			formattedPubKeys[i] = fmt.Sprintf("%#x", bytesutil.Trunc(pk[:]))
		}
		fmt.Printf("About to perform a voluntary exit of %d accounts\n", len(rawPubKeys))
	}

	promptHeader := au.Red("===============IMPORTANT===============")
	promptDescription := "Withdrawing funds is not possible in Phase 0 of the system. " +
		"Please navigate to the following website and make sure you understand the current implications " +
		"of a voluntary exit before making the final decision:"
	promptURL := au.Blue("https://docs.prylabs.network/docs/wallet/exiting-a-validator/#withdrawal-delay-warning")
	promptQuestion := "If you still want to continue with the voluntary exit, please input a phrase found at the end " +
		"of the page from the above URL"
	promptText := fmt.Sprintf("%s\n%s\n%s\n%s", promptHeader, promptDescription, promptURL, promptQuestion)
	resp, err := promptutil.ValidatePrompt(r, promptText, func(input string) error {
		return promptutil.ValidatePhrase(input, exitPassphrase)
	})
	if err != nil {
		return nil, nil, err
	}
	if strings.EqualFold(resp, "n") {
		return nil, nil, nil
	}

	return rawPubKeys, formattedPubKeys, nil
}

func prepareClients(cliCtx *cli.Context) (*ethpb.BeaconNodeValidatorClient, *ethpb.NodeClient, error) {
	dialOpts := client.ConstructDialOptions(
		cliCtx.Int(cmd.GrpcMaxCallRecvMsgSizeFlag.Name),
		cliCtx.String(flags.CertFlag.Name),
		cliCtx.Uint(flags.GrpcRetriesFlag.Name),
		cliCtx.Duration(flags.GrpcRetryDelayFlag.Name),
	)
	if dialOpts == nil {
		return nil, nil, errors.New("failed to construct dial options")
	}

	grpcHeaders := strings.Split(cliCtx.String(flags.GrpcHeadersFlag.Name), ",")
	cliCtx.Context = grpcutils.AppendHeaders(cliCtx.Context, grpcHeaders)

	conn, err := grpc.DialContext(cliCtx.Context, cliCtx.String(flags.BeaconRPCProviderFlag.Name), dialOpts...)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not dial endpoint %s", flags.BeaconRPCProviderFlag.Name)
	}
	validatorClient := ethpb.NewBeaconNodeValidatorClient(conn)
	nodeClient := ethpb.NewNodeClient(conn)
	return &validatorClient, &nodeClient, nil
}

func performExit(cliCtx *cli.Context, cfg performExitCfg) (rawExitedKeys [][]byte, formattedExitedKeys []string, err error) {
	var rawNotExitedKeys [][]byte
	for i, key := range cfg.rawPubKeys {
		if err := client.ProposeExit(cliCtx.Context, cfg.validatorClient, cfg.nodeClient, cfg.keymanager.Sign, key); err != nil {
			rawNotExitedKeys = append(rawNotExitedKeys, key)

			msg := err.Error()
			if strings.Contains(msg, blocks.ValidatorAlreadyExitedMsg) ||
				strings.Contains(msg, blocks.ValidatorCannotExitYetMsg) {
				log.Warningf("Could not perform voluntary exit for account %s: %s", cfg.formattedPubKeys[i], msg)
			} else {
				log.WithError(err).Errorf("voluntary exit failed for account %s", cfg.formattedPubKeys[i])
			}
		}
	}

	rawExitedKeys = make([][]byte, 0)
	formattedExitedKeys = make([]string, 0)
	for i, key := range cfg.rawPubKeys {
		found := false
		for _, notExited := range rawNotExitedKeys {
			if bytes.Equal(notExited, key) {
				found = true
				break
			}
		}
		if !found {
			rawExitedKeys = append(rawExitedKeys, key)
			formattedExitedKeys = append(formattedExitedKeys, cfg.formattedPubKeys[i])
		}
	}

	return rawExitedKeys, formattedExitedKeys, nil
}

func displayExitInfo(rawExitedKeys [][]byte, trimmedExitedKeys []string) {
	if len(rawExitedKeys) > 0 {
		urlFormattedPubKeys := make([]string, len(rawExitedKeys))
		for i, key := range rawExitedKeys {
			var baseUrl string
			if params.BeaconConfig().ConfigName == params.ConfigNames[params.Pyrmont] {
				baseUrl = "https://pyrmont.beaconcha.in/validator/"
			} else if params.BeaconConfig().ConfigName == params.ConfigNames[params.Prater] {
				baseUrl = "https://prater.beaconcha.in/validator/"
			} else {
				baseUrl = "https://beaconcha.in/validator/"
			}
			// Remove '0x' prefix
			urlFormattedPubKeys[i] = baseUrl + hexutil.Encode(key)[2:]
		}

		ifaceKeys := make([]interface{}, len(urlFormattedPubKeys))
		for i, k := range urlFormattedPubKeys {
			ifaceKeys[i] = k
		}

		info := fmt.Sprintf("Voluntary exit was successful for the accounts listed. "+
			"URLs where you can track each validator's exit:\n"+strings.Repeat("%s\n", len(ifaceKeys)), ifaceKeys...)

		log.WithField("publicKeys", strings.Join(trimmedExitedKeys, ", ")).Info(info)
	} else {
		log.Info("No successful voluntary exits")
	}
}

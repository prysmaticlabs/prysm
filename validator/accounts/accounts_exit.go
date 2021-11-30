package accounts

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	grpcutil "github.com/prysmaticlabs/prysm/api/grpc"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/io/prompt"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/userprompt"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/client"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// PerformExitCfg for account voluntary exits.
type PerformExitCfg struct {
	ValidatorClient  ethpb.BeaconNodeValidatorClient
	NodeClient       ethpb.NodeClient
	Keymanager       keymanager.IKeymanager
	RawPubKeys       [][]byte
	FormattedPubKeys []string
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
	if nodeClient == nil {
		return errors.New("could not prepare beacon node client")
	}
	syncStatus, err := (*nodeClient).GetSyncStatus(cliCtx.Context, &emptypb.Empty{})
	if err != nil {
		return err
	}
	if syncStatus == nil {
		return errors.New("could not get sync status")
	}

	if (*syncStatus).Syncing {
		return errors.New("could not perform exit: beacon node is syncing.")
	}

	cfg := PerformExitCfg{
		*validatorClient,
		*nodeClient,
		kManager,
		rawPubKeys,
		trimmedPubKeys,
	}
	rawExitedKeys, trimmedExitedKeys, err := PerformVoluntaryExit(cliCtx.Context, cfg)
	if err != nil {
		return err
	}
	displayExitInfo(rawExitedKeys, trimmedExitedKeys)

	return nil
}

// PerformVoluntaryExit uses gRPC clients to submit a voluntary exit message to a beacon node.
func PerformVoluntaryExit(
	ctx context.Context, cfg PerformExitCfg,
) (rawExitedKeys [][]byte, formattedExitedKeys []string, err error) {
	var rawNotExitedKeys [][]byte
	for i, key := range cfg.RawPubKeys {
		if err := client.ProposeExit(ctx, cfg.ValidatorClient, cfg.NodeClient, cfg.Keymanager.Sign, key); err != nil {
			rawNotExitedKeys = append(rawNotExitedKeys, key)

			msg := err.Error()
			if strings.Contains(msg, blocks.ValidatorAlreadyExitedMsg) ||
				strings.Contains(msg, blocks.ValidatorCannotExitYetMsg) {
				log.Warningf("Could not perform voluntary exit for account %s: %s", cfg.FormattedPubKeys[i], msg)
			} else {
				log.WithError(err).Errorf("voluntary exit failed for account %s", cfg.FormattedPubKeys[i])
			}
		}
	}

	rawExitedKeys = make([][]byte, 0)
	formattedExitedKeys = make([]string, 0)
	for i, key := range cfg.RawPubKeys {
		found := false
		for _, notExited := range rawNotExitedKeys {
			if bytes.Equal(notExited, key) {
				found = true
				break
			}
		}
		if !found {
			rawExitedKeys = append(rawExitedKeys, key)
			formattedExitedKeys = append(formattedExitedKeys, cfg.FormattedPubKeys[i])
		}
	}

	return rawExitedKeys, formattedExitedKeys, nil
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
			userprompt.SelectAccountsVoluntaryExitPromptText,
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
				resp, err := prompt.ValidatePrompt(
					r, fmt.Sprintf(promptText, au.BrightGreen(formattedPubKeys[0])), prompt.ValidateYesOrNo,
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
				resp, err := prompt.ValidatePrompt(r, promptText, prompt.ValidateYesOrNo)
				if err != nil {
					return nil, nil, err
				}
				if strings.EqualFold(resp, "n") {
					return nil, nil, nil
				}
			}
		}
	} else {
		rawPubKeys, formattedPubKeys = prepareAllKeys(validatingPublicKeys)
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
	resp, err := prompt.ValidatePrompt(r, promptText, func(input string) error {
		return prompt.ValidatePhrase(input, exitPassphrase)
	})
	if err != nil {
		return nil, nil, err
	}
	if strings.EqualFold(resp, "n") {
		return nil, nil, nil
	}

	return rawPubKeys, formattedPubKeys, nil
}

func prepareAllKeys(validatingKeys [][48]byte) (raw [][]byte, formatted []string) {
	raw = make([][]byte, len(validatingKeys))
	formatted = make([]string, len(validatingKeys))
	for i, pk := range validatingKeys {
		raw[i] = make([]byte, len(pk))
		copy(raw[i], pk[:])
		formatted[i] = fmt.Sprintf("%#x", bytesutil.Trunc(pk[:]))
	}
	return
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
	cliCtx.Context = grpcutil.AppendHeaders(cliCtx.Context, grpcHeaders)

	conn, err := grpc.DialContext(cliCtx.Context, cliCtx.String(flags.BeaconRPCProviderFlag.Name), dialOpts...)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not dial endpoint %s", flags.BeaconRPCProviderFlag.Name)
	}
	validatorClient := ethpb.NewBeaconNodeValidatorClient(conn)
	nodeClient := ethpb.NewNodeClient(conn)
	return &validatorClient, &nodeClient, nil
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

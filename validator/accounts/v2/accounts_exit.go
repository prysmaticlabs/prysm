package v2

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/client"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ExitAccountsCli performs a voluntary exit on one or more accounts.
func ExitAccountsCli(cliCtx *cli.Context, r io.Reader) error {
	wallet, err := OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*Wallet, error) {
		return nil, errors.New(
			"no wallet found, no accounts to exit",
		)
	})
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}

	keymanager, err := wallet.InitializeKeymanager(cliCtx.Context, false /* skip mnemonic confirm */)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	validatingPublicKeys, err := keymanager.FetchValidatingPublicKeys(cliCtx.Context)
	if err != nil {
		return err
	}
	if len(validatingPublicKeys) == 0 {
		return errors.New("wallet is empty, no accounts to perform voluntary exit")
	}
	// Allow the user to interactively select the accounts to exit or optionally
	// provide them via cli flags as a string of comma-separated, hex strings.
	filteredPubKeys, err := filterPublicKeysFromUserInput(
		cliCtx,
		flags.VoluntaryExitPublicKeysFlag,
		validatingPublicKeys,
		selectAccountsVoluntaryExitPromptText,
	)
	if err != nil {
		return errors.Wrap(err, "could not filter public keys for voluntary exit")
	}
	rawPublicKeys := make([][]byte, len(filteredPubKeys))
	formattedPubKeys := make([]string, len(filteredPubKeys))
	for i, pk := range filteredPubKeys {
		pubKeyBytes := pk.Marshal()
		rawPublicKeys[i] = pubKeyBytes
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
				return err
			}
			if strings.ToLower(resp) == "n" {
				return nil
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
				return err
			}
			if strings.ToLower(resp) == "n" {
				return nil
			}
		}
	}

	promptHeader := au.Red("===============IMPORTANT===============")
	promptDescription := "Withdrawing funds is not possible in Phase 0 of the system. " +
		"Please navigate to the following website and make sure you understand the current implications " +
		"of a voluntary exit before making the final decision:"
	promptURL := au.Blue("https://docs.prylabs.network/docs/faq/#can-i-get-back-my-testnet-eth-how-can-i-withdraw-my-validator-gains")
	promptQuestion := "Do you still want to continue with the voluntary exit? Y/N"
	promptText := fmt.Sprintf("%s\n%s\n%s\n%s", promptHeader, promptDescription, promptURL, promptQuestion)
	resp, err := promptutil.ValidatePrompt(r, promptText, promptutil.ValidateYesOrNo)
	if err != nil {
		return err
	}
	if strings.ToLower(resp) == "n" {
		return nil
	}

	dialOpts := client.ConstructDialOptions(
		cmd.GrpcMaxCallRecvMsgSizeFlag.Value,
		flags.CertFlag.Value,
		strings.Split(flags.GrpcHeadersFlag.Value, ","),
		flags.GrpcRetriesFlag.Value,
		flags.GrpcRetryDelayFlag.Value,
	)
	if dialOpts == nil {
		return errors.New("failed to construct dial options")
	}
	conn, err := grpc.DialContext(cliCtx.Context, cliCtx.String(flags.BeaconRPCProviderFlag.Name), dialOpts...)
	if err != nil {
		return errors.Wrapf(err, "could not dial endpoint %s", flags.BeaconRPCProviderFlag.Name)
	}
	validatorClient := ethpb.NewBeaconNodeValidatorClient(conn)

	var rawNotExitedKeys [][]byte
	for _, key := range rawPublicKeys {
		if err := client.ProposeExit(cliCtx.Context, validatorClient, keymanager, key); err != nil {
			rawNotExitedKeys = append(rawNotExitedKeys, key)
			log.WithError(err).Errorf("voluntary exit failed for account %s", key)
		}
	}
	var formattedExitedKeys []string
	for _, notExited := range rawNotExitedKeys {
		for i, key := range rawPublicKeys {
			if bytes.Equal(notExited, key) {
				formattedExitedKeys = append(formattedExitedKeys, formattedPubKeys[i])
				break
			}
		}
	}

	log.WithField("publicKeys", strings.Join(formattedExitedKeys, ", ")).
		Info("Voluntary exit was successful for the accounts listed")

	return nil
}

// ExitAccountsUnimplemented is a stub for ExitAccounts until the latter is fully implemented.
func ExitAccountsUnimplemented(cliCtx *cli.Context, r io.Reader) error {
	return status.Errorf(codes.Unimplemented, "method ExitAccounts not implemented")
}

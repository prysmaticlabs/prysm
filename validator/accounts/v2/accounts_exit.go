package v2

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ExitAccounts performs a voluntary exit on one or more accounts.
func ExitAccounts(cliCtx *cli.Context, stdin io.Reader) error {
	ctx := context.Background()
	wallet, err := OpenWallet(cliCtx)
	if errors.Is(err, ErrNoWalletFound) {
		return errors.Wrap(err, "no wallet found at path, create a new wallet with wallet-v2 create")
	} else if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	skipMnemonicConfirm := cliCtx.Bool(flags.SkipMnemonicConfirmFlag.Name)
	keymanager, err := wallet.InitializeKeymanager(cliCtx, skipMnemonicConfirm)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	validatingPublicKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
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
				stdin, fmt.Sprintf(promptText, au.BrightGreen(formattedPubKeys[0])), promptutil.ValidateYesOrNo,
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
			resp, err := promptutil.ValidatePrompt(stdin, promptText, promptutil.ValidateYesOrNo)
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
	resp, err := promptutil.ValidatePrompt(stdin, promptText, promptutil.ValidateYesOrNo)
	if err != nil {
		return err
	}
	if strings.ToLower(resp) == "n" {
		return nil
	}

	log.WithField("publicKeys", allAccountStr).Info("Voluntary exit was successful")
	return nil
}

// ExitAccountsUnimplemented is a stub for ExitAccounts until the latter is fully implemented.
func ExitAccountsUnimplemented(cliCtx *cli.Context, stdin io.Reader) error {
	return status.Errorf(codes.Unimplemented, "method ExitAccounts not implemented")
}

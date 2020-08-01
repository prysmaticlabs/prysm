package v2

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
)

// ListAccounts displays all available validator accounts in a Prysm wallet.
func ListAccounts(cliCtx *cli.Context) error {
	// Read the wallet from the specified path.
	ctx := context.Background()
	wallet, err := OpenWallet(cliCtx)
	if errors.Is(err, ErrNoWalletFound) {
		return errors.Wrap(err, "no wallet found at path, create a new wallet with wallet-v2 create")
	} else if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	keymanager, err := wallet.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	showDepositData := cliCtx.Bool(flags.ShowDepositDataFlag.Name)
	switch wallet.KeymanagerKind() {
	case v2keymanager.Direct:
		km, ok := keymanager.(*direct.Keymanager)
		if !ok {
			return errors.New("could not assert keymanager interface to concrete type")
		}
		if err := listDirectKeymanagerAccounts(showDepositData, wallet, km); err != nil {
			return errors.Wrap(err, "could not list validator accounts with direct keymanager")
		}
	case v2keymanager.Derived:
		km, ok := keymanager.(*derived.Keymanager)
		if !ok {
			return errors.New("could not assert keymanager interface to concrete type")
		}
		if err := listDerivedKeymanagerAccounts(showDepositData, wallet, km); err != nil {
			return errors.Wrap(err, "could not list validator accounts with derived keymanager")
		}
	case v2keymanager.Remote:
		km, ok := keymanager.(*remote.Keymanager)
		if !ok {
			return errors.New("could not assert keymanager interface to concrete type")
		}
		if err := listRemoteKeymanagerAccounts(wallet, km, km.Config()); err != nil {
			return errors.Wrap(err, "could not list validator accounts with remote keymanager")
		}
	default:
		return fmt.Errorf("keymanager kind %s not yet supported", wallet.KeymanagerKind().String())
	}
	return nil
}

func listDirectKeymanagerAccounts(
	showDepositData bool,
	wallet *Wallet,
	keymanager *direct.Keymanager,
) error {
	// We initialize the wallet's keymanager.
	accountNames, err := keymanager.ValidatingAccountNames()
	if err != nil {
		return errors.Wrap(err, "could not fetch account names")
	}
	au := aurora.NewAurora(true)
	numAccounts := au.BrightYellow(len(accountNames))
	fmt.Println("")
	if len(accountNames) == 1 {
		fmt.Printf("Showing %d validator account\n", numAccounts)
	} else {
		fmt.Printf("Showing %d validator accounts\n", numAccounts)
	}
	fmt.Println(
		au.BrightRed("View the eth1 deposit transaction data for your accounts " +
			"by running `validator accounts-v2 list --show-deposit-data"),
	)

	ctx := context.Background()
	pubKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch validating public keys")
	}
	for i := 0; i < len(accountNames); i++ {
		fmt.Println("")
		fmt.Printf("%s | %s\n", au.BrightBlue(fmt.Sprintf("Account %d", i)).Bold(), au.BrightGreen(accountNames[i]).Bold())
		fmt.Printf("%s %#x\n", au.BrightMagenta("[validating public key]").Bold(), pubKeys[i])
		if !showDepositData {
			continue
		}
		fmt.Printf(
			"%s\n",
			au.BrightRed("If you imported your account coming from the eth2 launchpad, you will find your "+
				"deposit_data.json in the eth2.0-deposit-cli's validator_keys folder"),
		)
		fmt.Println("")
	}
	fmt.Println("")
	return nil
}

func listDerivedKeymanagerAccounts(
	showDepositData bool,
	wallet *Wallet,
	keymanager *derived.Keymanager,
) error {
	au := aurora.NewAurora(true)
	fmt.Println(
		au.BrightRed("View the eth1 deposit transaction data for your accounts " +
			"by running `validator accounts-v2 list --show-deposit-data"),
	)
	fmt.Printf("(keymanager kind) %s\n", au.BrightGreen("derived, (HD) hierarchical-deterministic").Bold())
	fmt.Printf("(derivation format) %s\n", au.BrightGreen(keymanager.Config().DerivedPathStructure).Bold())
	ctx := context.Background()
	validatingPubKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch validating public keys")
	}
	withdrawalPublicKeys, err := keymanager.FetchWithdrawalPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch validating public keys")
	}
	nextAccountNumber := keymanager.NextAccountNumber(ctx)
	currentAccountNumber := nextAccountNumber
	if nextAccountNumber > 0 {
		currentAccountNumber--
	}
	accountNames, err := keymanager.ValidatingAccountNames(ctx)
	if err != nil {
		return err
	}
	if len(accountNames) == 1 {
		fmt.Print("Showing 1 validator account\n")
	} else if len(accountNames) == 0 {
		fmt.Print("No accounts found\n")
		return nil
	} else {
		fmt.Printf("Showing %d validator accounts\n", len(accountNames))
	}
	for i := uint64(0); i <= currentAccountNumber; i++ {
		fmt.Println("")
		validatingKeyPath := fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, i)
		withdrawalKeyPath := fmt.Sprintf(derived.WithdrawalKeyDerivationPathTemplate, i)

		// Retrieve the withdrawal key account metadata.
		fmt.Printf("%s | %s\n", au.BrightBlue(fmt.Sprintf("Account %d", i)).Bold(), au.BrightGreen(accountNames[i]).Bold())
		fmt.Printf("%s %#x\n", au.BrightMagenta("[withdrawal public key]").Bold(), withdrawalPublicKeys[i])
		fmt.Printf("%s %s\n", au.BrightMagenta("[derivation path]").Bold(), withdrawalKeyPath)

		// Retrieve the validating key account metadata.
		fmt.Printf("%s %#x\n", au.BrightCyan("[validating public key]").Bold(), validatingPubKeys[i])
		fmt.Printf("%s %s\n", au.BrightCyan("[derivation path]").Bold(), validatingKeyPath)

		if !showDepositData {
			continue
		}
		enc, err := keymanager.DepositDataForAccount(i)
		if err != nil {
			return errors.Wrapf(err, "could not deposit data for account: %s", accountNames[i])
		}
		fmt.Printf(`
======================SSZ Deposit Data=====================

%#x

===================================================================`, enc)
		fmt.Println(" ")
	}
	return nil
}

func listRemoteKeymanagerAccounts(
	wallet *Wallet,
	keymanager v2keymanager.IKeymanager,
	cfg *remote.Config,
) error {
	au := aurora.NewAurora(true)
	fmt.Printf("(keymanager kind) %s\n", au.BrightGreen("remote signer").Bold())
	fmt.Printf(
		"(configuration file path) %s\n",
		au.BrightGreen(filepath.Join(wallet.AccountsDir(), KeymanagerConfigFileName)).Bold(),
	)
	ctx := context.Background()
	fmt.Println(" ")
	fmt.Printf("%s\n", au.BrightGreen("Configuration options").Bold())
	fmt.Println(cfg)
	validatingPubKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch validating public keys")
	}
	if len(validatingPubKeys) == 1 {
		fmt.Print("Showing 1 validator account\n")
	} else if len(validatingPubKeys) == 0 {
		fmt.Print("No accounts found\n")
		return nil
	} else {
		fmt.Printf("Showing %d validator accounts\n", len(validatingPubKeys))
	}
	for i := 0; i < len(validatingPubKeys); i++ {
		fmt.Println("")
		fmt.Printf(
			"%s\n", au.BrightGreen(petnames.DeterministicName(validatingPubKeys[i][:], "-")).Bold(),
		)
		// Retrieve the validating key account metadata.
		fmt.Printf("%s %#x\n", au.BrightCyan("[validating public key]").Bold(), validatingPubKeys[i])
		fmt.Println(" ")
	}
	return nil
}

package v2

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	"github.com/urfave/cli/v2"
)

// ListAccounts displays all available validator accounts in a Prysm wallet.
func ListAccounts(cliCtx *cli.Context) error {
	// Read the wallet from the specified path.
	ctx := context.Background()
	wallet, err := OpenWallet(cliCtx)
	if err != nil {
		return errors.Wrapf(err, "could not read wallet at specified path %s", wallet.AccountsDir())
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

		// Retrieve the account creation timestamp.
		createdAtBytes, err := wallet.ReadFileAtPath(ctx, accountNames[i], direct.TimestampFileName)
		if err != nil {
			return errors.Wrapf(err, "could not read file for account: %s", direct.TimestampFileName)
		}
		unixTimestampStr, err := strconv.ParseInt(string(createdAtBytes), 10, 64)
		if err != nil {
			return errors.Wrapf(err, "could not parse account created at timestamp: %s", createdAtBytes)
		}
		unixTimestamp := time.Unix(unixTimestampStr, 0)

		fmt.Printf("%s | %s\n", au.BrightGreen(accountNames[i]).Bold(), humanize.Time(unixTimestamp))
		fmt.Printf("%s %#x\n", au.BrightMagenta("[validating public key]").Bold(), pubKeys[i])
		if !showDepositData {
			continue
		}
		enc, err := wallet.ReadFileAtPath(ctx, accountNames[i], direct.DepositTransactionFileName)
		if err != nil {
			return errors.Wrapf(err, "could not read file for account: %s", direct.DepositTransactionFileName)
		}
		fmt.Printf(
			"%s %s\n",
			"(deposit tx file)",
			filepath.Join(wallet.AccountsDir(), accountNames[i], direct.DepositTransactionFileName),
		)
		fmt.Printf(`
======================Deposit Transaction Data=====================

%#x

===================================================================`, enc)
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
		createdAtBytes, err := wallet.ReadFileAtPath(ctx, validatingKeyPath, derived.TimestampFileName)
		if err != nil {
			return errors.Wrapf(err, "could not read file for account: %s", derived.TimestampFileName)
		}
		unixTimestampInt, err := strconv.ParseInt(string(createdAtBytes), 10, 64)
		if err != nil {
			return errors.Wrapf(err, "could not parse account created at timestamp: %s", createdAtBytes)
		}
		unixTimestamp := time.Unix(unixTimestampInt, 0)
		fmt.Printf("%s | %s\n", au.BrightGreen(accountNames[i]).Bold(), humanize.Time(unixTimestamp))
		fmt.Printf("%s %#x\n", au.BrightMagenta("[withdrawal public key]").Bold(), withdrawalPublicKeys[i])
		fmt.Printf("%s %s\n", au.BrightMagenta("[derivation path]").Bold(), withdrawalKeyPath)

		// Retrieve the validating key account metadata.
		fmt.Printf("%s %#x\n", au.BrightCyan("[validating public key]").Bold(), validatingPubKeys[i])
		fmt.Printf("%s %s\n", au.BrightCyan("[derivation path]").Bold(), validatingKeyPath)

		if !showDepositData {
			continue
		}
		enc, err := wallet.ReadFileAtPath(ctx, withdrawalKeyPath, derived.DepositTransactionFileName)
		if err != nil {
			return errors.Wrapf(err, "could not read file for account: %s", direct.DepositTransactionFileName)
		}
		fmt.Printf(
			"%s %s\n",
			"(deposit tx file)",
			filepath.Join(wallet.AccountsDir(), withdrawalKeyPath, derived.DepositTransactionFileName),
		)
		fmt.Printf(`
======================Deposit Transaction Data=====================

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

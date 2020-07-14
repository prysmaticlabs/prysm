package v2

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/urfave/cli/v2"
)

// ListAccounts displays all available validator accounts in a Prysm wallet.
func ListAccounts(cliCtx *cli.Context) error {
	walletDir, err := inputWalletDir(cliCtx)
	if err != nil {
		log.Fatal(err)
	}
	// Read the wallet from the specified path.
	ctx := context.Background()
	wallet, err := OpenWallet(ctx, &WalletConfig{
		WalletDir:         walletDir,
		CanUnlockAccounts: false,
	})
	if err != nil {
		log.Fatalf("Could not read wallet at specified path %s: %v", walletDir, err)
	}
	keymanager, err := wallet.InitializeKeymanager(ctx)
	if err != nil {
		log.Fatalf("Could not initialize keymanager: %v", err)
	}
	showDepositData := cliCtx.Bool(flags.ShowDepositDataFlag.Name)
	switch wallet.KeymanagerKind() {
	case v2keymanager.Direct:
		if err := listDirectKeymanagerAccounts(showDepositData, wallet, keymanager); err != nil {
			log.Fatalf("Could not list validator accounts with direct keymanager: %v", err)
		}
	default:
		log.Fatalf("Keymanager kind %s not yet supported", wallet.KeymanagerKind().String())
	}
	return nil
}

func listDirectKeymanagerAccounts(
	showDepositData bool,
	wallet *Wallet,
	keymanager v2keymanager.IKeymanager,
) error {
	// We initialize the wallet's keymanager.
	accountNames, err := wallet.AccountNames()
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
	dirPath := au.BrightCyan("(wallet dir)")
	fmt.Printf("%s %s\n", dirPath, wallet.AccountsDir())
	fmt.Printf("Keymanager kind: %s\n", au.BrightGreen(wallet.KeymanagerKind().String()).Bold())

	pubKeys, err := keymanager.FetchValidatingPublicKeys(context.Background())
	if err != nil {
		return errors.Wrap(err, "could not fetch validating public keys")
	}
	for i := 0; i < len(accountNames); i++ {
		fmt.Println("")
		fmt.Printf("%s\n", au.BrightGreen(accountNames[i]).Bold())
		fmt.Printf("%s %#x\n", au.BrightMagenta("[public key]").Bold(), pubKeys[i])

		// Retrieve the account creation timestamp.
		createdAtBytes, err := wallet.ReadFileForAccount(accountNames[i], direct.TimestampFileName)
		if err != nil {
			return errors.Wrapf(err, "could not read file for account: %s", direct.TimestampFileName)
		}
		unixTimestamp, err := strconv.ParseInt(string(createdAtBytes), 10, 64)
		if err != nil {
			return errors.Wrapf(err, "could not parse account created at timestamp: %s", createdAtBytes)
		}
		unixTimestampStr := time.Unix(unixTimestamp, 0)
		fmt.Printf("%s %v\n", au.BrightCyan("[created at]").Bold(), unixTimestampStr.String())
		if !showDepositData {
			continue
		}
		enc, err := wallet.ReadFileForAccount(accountNames[i], direct.DepositTransactionFileName)
		if err != nil {
			return errors.Wrapf(err, "could not read file for account: %s", direct.DepositTransactionFileName)
		}
		fmt.Printf(
			"%s %s\n",
			"(deposit tx file)",
			path.Join(wallet.AccountsDir(), accountNames[i], direct.DepositTransactionFileName),
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

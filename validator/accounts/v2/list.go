package v2

import (
	"context"
	"fmt"
	"path"

	"github.com/logrusorgru/aurora"
	"github.com/urfave/cli/v2"

	"github.com/prysmaticlabs/prysm/validator/flags"
)

// ListAccounts --
func ListAccounts(cliCtx *cli.Context) error {
	walletDir := cliCtx.String(flags.WalletDirFlag.Name)
	if walletDir == flags.DefaultValidatorDir() {
		walletDir = path.Join(walletDir, WalletDefaultDirName)
	}
	passwordsDir := cliCtx.String(flags.WalletPasswordsDirFlag.Name)
	if passwordsDir == flags.DefaultValidatorDir() {
		passwordsDir = path.Join(passwordsDir, PasswordsDefaultDirName)
	}
	// Read the wallet from the specified path.
	ctx := context.Background()
	wallet, err := OpenWallet(ctx, &WalletConfig{
		PasswordsDir: passwordsDir,
		WalletDir:    walletDir,
	})
	if err == ErrNoWalletFound {
		log.Fatal("No wallet found")
	} else if err != nil {
		log.Fatalf("Could not read wallet at specified path %s: %v", walletDir, err)
	}
	// We initialize the wallet's keymanager.
	accountNames, err := wallet.AccountNames()
	if err != nil {
		log.Fatal(err)
	}
	keymanager, err := wallet.ExistingKeyManager(ctx)
	if err != nil {
		log.Fatalf("Could not initialize keymanager: %v", err)
	}
	pubKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		log.Fatal(err)
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
			"by running `validator accounts-v2 list --deposit-data"),
	)
	dirPath := au.BrightCyan("(wallet dir)")
	fmt.Printf("%s %s\n", dirPath, wallet.AccountsDir())
	dirPath = au.BrightCyan("(passwords dir)")
	fmt.Printf("%s %s\n", dirPath, wallet.passwordsDir)
	fmt.Printf("Keymanager kind: %s\n", au.BrightGreen(wallet.KeymanagerKind().String()).Bold())

	showDepositData := cliCtx.Bool(flags.ShowDepositDataFlag.Name)
	for i := 0; i < len(accountNames); i++ {
		fmt.Println("")
		fmt.Printf("%s\n", au.BrightGreen(accountNames[i]).Bold())
		fmt.Printf("%s %#x\n", au.BrightMagenta("[public key]").Bold(), pubKeys[i])
		fmt.Printf("%s %s\n", au.BrightCyan("[created at]").Bold(), "July 07, 2020 2:32 PM")
		fmt.Printf("%s %s\n", "(eth1 tx data file)", "deposit_transaction.rlp")
		if !showDepositData {
			continue
		}
		enc, err := wallet.ReadFileForAccount(accountNames[i], "deposit_transaction.rlp")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(`
======================Deposit Transaction Data=====================

%#x

===================================================================`, enc)
		fmt.Println("")
		fmt.Println(
			au.BrightRed("Enter the above deposit data into step 3 on https://prylabs.net/participate").Bold(),
		)
	}
	fmt.Println("")
	return nil
}

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
		fmt.Printf("Found %d validator account\n", numAccounts)
	} else {
		fmt.Printf("Found %d validator accounts\n", numAccounts)
	}
	fmt.Printf("%s %s\n", dirPath, wallet.AccountsDir())
	dirPath := au.BrightMagenta("(directory path)")
	fmt.Printf("%s %s\n", dirPath, wallet.AccountsDir())
	passwordsPath := au.BrightCyan("(passwords path)")
	fmt.Printf("%s %s\n", passwordsPath, wallet.passwordsDir)
	for i := 0; i < len(accountNames); i++ {
		fmt.Println("")
		fmt.Println(au.BrightGreen(accountNames[i]).Bold())
		fmt.Printf("%#x\n", pubKeys[i])
	}
	return nil
}

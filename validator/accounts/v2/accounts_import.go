package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/prysmaticlabs/prysm/shared/petnames"

	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/urfave/cli/v2"
)

// ImportAccount uses the archived account made from ExportAccount to import an account and
// asks the users for account passwords.
func ImportAccount(cliCtx *cli.Context) error {
	walletDir, err := inputDirectory(cliCtx, walletDirPromptText, flags.WalletDirFlag)
	if err != nil && !errors.Is(err, ErrNoWalletFound) {
		return errors.Wrap(err, "could not parse wallet directory")
	}
	passwordsDir, err := inputDirectory(cliCtx, passwordsDirPromptText, flags.WalletPasswordsDirFlag)
	if err != nil {
		return err
	}
	keysDir, err := inputDirectory(cliCtx, keysDirPromptText, flags.KeysDirFlag)
	if err != nil {
		return errors.Wrap(err, "could not parse output directory")
	}

	accountsPath := filepath.Join(walletDir, v2keymanager.Direct.String())
	if err := os.MkdirAll(accountsPath, DirectoryPermissions); err != nil {
		return errors.Wrap(err, "could not create wallet directory")
	}
	if err := os.MkdirAll(passwordsDir, DirectoryPermissions); err != nil {
		return errors.Wrap(err, "could not create passwords directory")
	}

	wallet := &Wallet{
		accountsPath:   accountsPath,
		passwordsDir:   passwordsDir,
		keymanagerKind: v2keymanager.Direct,
	}

	var accountsImported []string
	ctx := context.Background()
	if err := filepath.Walk(keysDir, func(path string, info os.FileInfo, err error) error {
		keystoreBytes, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "could not read keystore file")
		}
		keystoreFile := &v2keymanager.Keystore{}
		if err := json.Unmarshal(keystoreBytes, keystoreFile); err != nil {
			return errors.Wrap(err, "could not decode keystore json")
		}

		fileName := filepath.Base(path)
		if !info.IsDir() && !strings.Contains(fileName, "keystore") {
			return nil
		}
		accountName, err := wallet.importKeystore(ctx, path)
		if err != nil {
			return errors.Wrap(err, "could not import keystore")
		}
		if err := wallet.enterPasswordForAccount(cliCtx, accountName); err != nil {
			return errors.Wrap(err, "could not verify password for keystore")
		}
		accountsImported = append(accountsImported, accountName)
		return nil
	}); err != nil {
		return errors.Wrap(err, "could not walk files")
	}
	au := aurora.NewAurora(true)
	var loggedAccounts []string
	for _, accountName := range accountsImported {
		loggedAccounts = append(loggedAccounts, fmt.Sprintf("%s", au.BrightGreen(accountName).Bold()))
	}

	keymanager, err := wallet.InitializeKeymanager(context.Background(), true /* skip mnemonic confirm */)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	km, ok := keymanager.(*direct.Keymanager)
	if !ok {
		return errors.New("can only export accounts for a non-HD wallet")
	}
	if err := logAccountsImported(wallet, km, accountsImported); err != nil {
		return errors.Wrap(err, "could not log accounts imported")
	}

	return nil
}

func (w *Wallet) importKeystore(ctx context.Context, keystoreFilePath string) (string, error) {
	keystoreBytes, err := ioutil.ReadFile(keystoreFilePath)
	if err != nil {
		return "", errors.Wrap(err, "could not read keystore file")
	}
	keystoreFile := &v2keymanager.Keystore{}
	if err := json.Unmarshal(keystoreBytes, keystoreFile); err != nil {
		return "", errors.Wrap(err, "could not decode keystore json")
	}
	accountName := petnames.DeterministicName([]byte(keystoreFile.Pubkey), "-")
	keystoreFileName := filepath.Base(keystoreFilePath)
	if err := w.WriteFileAtPath(ctx, accountName, keystoreFileName, keystoreBytes); err != nil {
		return "", errors.Wrap(err, "could not write keystore to account dir")
	}
	return accountName, nil
}

func logAccountsImported(wallet *Wallet, keymanager *direct.Keymanager, accountNames []string) error {
	au := aurora.NewAurora(true)

	numAccounts := au.BrightYellow(len(accountNames))
	fmt.Println("")
	if len(accountNames) == 1 {
		fmt.Printf("Imported %d validator account\n", numAccounts)
	} else {
		fmt.Printf("Imported %d validator accounts\n", numAccounts)
	}
	for _, accountName := range accountNames {
		fmt.Println("")
		fmt.Printf("%s\n", au.BrightGreen(accountName).Bold())

		publicKey, err := keymanager.PublicKeyForAccount(accountName)
		if err != nil {
			return errors.Wrap(err, "could not get public key")
		}
		fmt.Printf("%s %#x\n", au.BrightMagenta("[public key]").Bold(), publicKey)

		dirPath := au.BrightCyan("(wallet dir)")
		fmt.Printf("%s %s\n", dirPath, filepath.Join(wallet.AccountsDir(), accountName))
	}
	return nil
}

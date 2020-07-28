package v2

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/urfave/cli/v2"
)

const allAccountsText = "All accounts"
const archiveFilename = "backup.zip"

// ExportAccount creates a zip archive of the selected accounts to be used in the future for importing accounts.
func ExportAccount(cliCtx *cli.Context) error {
	outputDir, err := inputDirectory(cliCtx, exportDirPromptText, flags.BackupDirFlag)
	if err != nil {
		return errors.Wrap(err, "could not parse output directory")
	}
	wallet, err := OpenWallet(cliCtx)
	if errors.Is(err, ErrNoWalletFound) {
		return errors.Wrap(err, "no wallet found at path, create a new wallet with wallet-v2 create")
	} else if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	keymanager, err := wallet.InitializeKeymanager(context.Background(), true /* skip mnemonic confirm */)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	km, ok := keymanager.(*direct.Keymanager)
	if !ok {
		return errors.New("can only export accounts for a non-HD wallet")
	}
	allAccounts, err := km.ValidatingAccountNames()
	if err != nil {
		return errors.Wrap(err, "could not get account names")
	}
	accounts, err := selectAccounts(cliCtx, allAccounts)
	if err != nil {
		return errors.Wrap(err, "could not select accounts")
	}
	if len(accounts) == 0 {
		return errors.New("no accounts to export")
	}

	if err := wallet.zipAccounts(accounts, outputDir); err != nil {
		return errors.Wrap(err, "could not export accounts")
	}

	if err := logAccountsExported(wallet, km, accounts); err != nil {
		return errors.Wrap(err, "could not log out exported accounts")
	}

	return nil
}

func selectAccounts(cliCtx *cli.Context, accounts []string) ([]string, error) {
	if len(accounts) == 1 {
		return accounts, nil
	}
	if cliCtx.IsSet(flags.AccountsFlag.Name) {
		enteredAccounts := cliCtx.StringSlice(flags.AccountsFlag.Name)
		if len(enteredAccounts) == 1 && enteredAccounts[0] == "all" {
			return accounts, nil
		}
		allAccountsStr := strings.Join(accounts, " ")
		for _, accountName := range enteredAccounts {
			if !strings.Contains(allAccountsStr, accountName) {
				return nil, fmt.Errorf("entered account %s not found in given wallet directory", accountName)
			}
		}
		return enteredAccounts, nil
	}
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "\U0001F336 {{ .Name | cyan }}",
		Inactive: "  {{ .Name | cyan }}",
		Selected: "\U0001F336 {{ .Name | red | cyan }}",
		Details: `
--------- Account ----------
{{ "Name:" | faint }}	{{ .Name }}`,
	}

	var result string
	var err error
	exit := "Exit Account Selection"
	results := []string{}
	au := aurora.NewAurora(true)
	// Alphabetical Sort of accounts.
	sort.Strings(accounts)

	for result != exit {
		prompt := promptui.Select{
			Label:        "Select accounts to backup",
			HideSelected: true,
			Items:        append([]string{exit, allAccountsText}, accounts...),
			Templates:    templates,
		}

		_, result, err = prompt.Run()
		if err != nil {
			return nil, err
		}
		if result == exit {
			fmt.Printf("%s\n", au.BrightRed("Exiting Selection").Bold())
			return results, nil
		}
		if result == allAccountsText {
			fmt.Printf("%s\n", au.BrightRed("[Selected all accounts]").Bold())
			return accounts, nil
		}
		results = append(results, result)
		fmt.Printf("%s %s\n", au.BrightRed("[Selected Account Name]").Bold(), result)
	}

	return results, nil
}

func (w *Wallet) zipAccounts(accounts []string, targetPath string) error {
	sourcePath := filepath.Dir(w.accountsPath)
	archivePath := filepath.Join(targetPath, archiveFilename)
	if err := os.MkdirAll(targetPath, params.BeaconIoConfig().ReadWriteExecutePermissions); err != nil {
		return errors.Wrap(err, "could not create target folder")
	}
	if fileExists(archivePath) {
		return errors.Errorf("Zip file already exists in directory: %s", archivePath)
	}
	zipfile, err := os.Create(archivePath)
	if err != nil {
		return errors.Wrap(err, "could not create zip file")
	}
	defer func() {
		if err := zipfile.Close(); err != nil {
			log.WithError(err).Error("Could not close zipfile")
		}
	}()

	archive := zip.NewWriter(zipfile)
	defer func() {
		if err := archive.Close(); err != nil {
			log.WithError(err).Error("Could not close archive")
		}
	}()

	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "could not walk %s", sourcePath)
		}

		var isAccount bool
		for _, accountName := range accounts {
			if strings.Contains(path, accountName) {
				// Add all files under the account folder to the archive.
				isAccount = true
			} else if !info.IsDir() && info.Name() == flags.KeymanagerConfigFileName {
				// Add the keymanager config file to the archive as well.
				isAccount = true
			}
		}
		if !isAccount {
			return nil
		}

		return copyFileFromZip(archive, sourcePath, info, path)
	})
	if err != nil {
		return errors.Wrap(err, "could not walk files")
	}
	return nil
}

func copyFileFromZip(archive *zip.Writer, sourcePath string, info os.FileInfo, path string) error {
	sourceDir := filepath.Base(sourcePath)
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return errors.Wrap(err, "could not get zip file info header")
	}
	if sourceDir != "" {
		header.Name = filepath.Join(sourceDir, strings.TrimPrefix(path, sourcePath))
	}
	if info.IsDir() {
		header.Name += "/"
	} else {
		header.Method = zip.Deflate
	}

	writer, err := archive.CreateHeader(header)
	if err != nil {
		return errors.Wrap(err, "could not create header")
	}

	if info.IsDir() {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.WithError(err).Error("Could not close file")
		}
	}()
	_, err = io.Copy(writer, file)
	return err
}

func logAccountsExported(wallet *Wallet, keymanager *direct.Keymanager, accountNames []string) error {
	au := aurora.NewAurora(true)

	numAccounts := au.BrightYellow(len(accountNames))
	fmt.Println("")
	if len(accountNames) == 1 {
		fmt.Printf("Exported %d validator account\n", numAccounts)
	} else {
		fmt.Printf("Exported %d validator accounts\n", numAccounts)
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

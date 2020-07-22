package v2

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

const allAccountsText = "All accounts"
const archiveFilename = "backup.zip"

// ExportAccount creates a zip archive of the selected accounts to be used in the future for importing accounts.
func ExportAccount(cliCtx *cli.Context) error {
	outputDir, err := inputExportDir(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not parse output directory")
	}

	wallet, err := OpenWallet(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}

	allAccounts, err := wallet.AccountNames()
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

	if err := logAccountsExported(wallet, accounts); err != nil {
		return errors.Wrap(err, "could not log out exported accounts")
	}

	return nil
}

func inputExportDir(cliCtx *cli.Context) (string, error) {
	outputDir := cliCtx.String(flags.BackupPathFlag.Name)
	if cliCtx.IsSet(flags.BackupPathFlag.Name) {
		return outputDir, nil
	}
	if outputDir == flags.DefaultValidatorDir() {
		outputDir = path.Join(outputDir)
	}
	prompt := promptui.Prompt{
		Label:    "Enter a file location to write the exported wallet to",
		Validate: validateDirectoryPath,
		Default:  outputDir,
	}
	outputPath, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("could not determine output directory: %v", formatPromptError(err))
	}
	return outputPath, nil
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

	prompt := promptui.SelectWithAdd{
		Label: "Select accounts to backup",
		Items: append(accounts, allAccountsText),
	}

	_, result, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	if result == allAccountsText {
		return accounts, nil
	}
	return []string{result}, nil
}

func (w *Wallet) zipAccounts(accounts []string, targetPath string) error {
	sourcePath := w.accountsPath
	archivePath := filepath.Join(targetPath, archiveFilename)
	if err := os.MkdirAll(targetPath, params.BeaconIoConfig().ReadWriteExecutePermissions); err != nil {
		return errors.Wrap(err, "could not create target folder")
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
			return errors.Wrap(err, "could not walk")
		}

		var isAccount bool
		for _, accountName := range accounts {
			if strings.Contains(path, accountName) {
				// Add all files under the account folder to the archive.
				isAccount = true
			} else if !info.IsDir() && info.Name() == KeymanagerConfigFileName {
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

func logAccountsExported(wallet *Wallet, accountNames []string) error {
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

		publicKey, err := wallet.publicKeyForAccount(accountName)
		if err != nil {
			return errors.Wrap(err, "could not get public key")
		}
		fmt.Printf("%s %#x\n", au.BrightMagenta("[public key]").Bold(), publicKey)

		dirPath := au.BrightCyan("(wallet dir)")
		fmt.Printf("%s %s\n", dirPath, filepath.Join(wallet.AccountsDir(), accountName))
	}
	return nil
}

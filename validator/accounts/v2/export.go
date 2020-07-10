package v2

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/manifoldco/promptui"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

const allAccountsText = "All accounts"
const archiveFilename = "backup.zip"

// ExportAccount creates a zip archive of the selected accounts to be used in the future for importing accounts.
func ExportAccount(cliCtx *cli.Context) error {
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not parse wallet directory")
	}

	outputDir, err := inputExportDir(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not parse output directory")
	}

	wallet, err := OpenWallet(context.Background(), &WalletConfig{
		CanUnlockAccounts: false,
		WalletDir:         walletDir,
	})
	if err == ErrNoWalletFound {
		return errors.New("no wallet found at path, please create a new wallet using `validator accounts-v2 new`")
	}
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}

	allAccounts, err := wallet.AccountNames()
	if err != nil {
		return err
	}
	accounts, err := selectAccounts(allAccounts)
	if err != nil {
		return err
	}

	if err := wallet.zipAccounts(accounts, outputDir); err != nil {
		return errors.Wrap(err, "could not zip accounts")
	}

	accountsBackedUp := accounts[0]
	if len(accounts) > 1 {
		accountsBackedUp = strings.Join(accounts, ", ")
	}
	backupFile := filepath.Join(outputDir, archiveFilename)
	log.WithField("exportPath", backupFile).Infof("Successfully exported account(s) %s", accountsBackedUp)
	return nil
}

func inputExportDir(cliCtx *cli.Context) (string, error) {
	outputDir := cliCtx.String(flags.BackupPathFlag.Name)
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

func selectAccounts(accounts []string) ([]string, error) {
	if len(accounts) == 1 {
		return accounts, nil
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

	baseDir := filepath.Base(sourcePath)

	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "could not walk")
		}

		var isAccount bool
		for _, accountName := range accounts {
			if strings.Contains(path, accountName) {
				// Add all files under the account folder to the archive.
				isAccount = true
			} else if !info.IsDir() && info.Name() == keymanagerConfigFileName {
				// Add the keymanager config file to the archive as well.
				isAccount = true
			}
		}
		if !isAccount {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return errors.Wrap(err, "could not get zip file info header")
		}
		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, sourcePath))
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
	})
	if err != nil {
		return errors.Wrap(err, "could not walk files")
	}
	return nil
}

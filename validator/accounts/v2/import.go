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

	"github.com/manifoldco/promptui"
	"github.com/prysmaticlabs/prysm/validator/flags"

	"github.com/pkg/errors"

	"github.com/urfave/cli/v2"
)

// ImportAccount uses the archived account made from ExportAccount to import an account and
// asks the users for account passwords.
func ImportAccount(cliCtx *cli.Context) error {
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if err != nil {
		log.Fatalf("Could not parse wallet directory: %v", err)
	}

	backupDir, err := inputImportDir(cliCtx)
	if err != nil {
		log.Fatalf("Could not parse output directory: %v", err)
	}

	if err := unzipArchiveToTarget(backupDir, walletDir); err != nil {
		log.WithError(err).Fatal("Could not unzip archive")
	}

	// Read the directory for password storage from user input.
	passwordsDirPath := inputPasswordsDirectory(cliCtx)

	wallet, err := CreateWallet(context.Background(), &WalletConfig{
		CanUnlockAccounts: true,
		PasswordsDir:      passwordsDirPath,
		WalletDir:         walletDir,
	})
	if err == ErrNoWalletFound {
		log.Fatal("No wallet found at path, please create a new wallet using `validator accounts-v2 new`")
	}
	if err != nil {
		log.Fatalf("Could not open wallet: %v", err)
	}

	accounts, err := wallet.AccountNames()
	if err != nil {
		return err
	}
	for _, accountName := range accounts {
		if err := wallet.enterPasswordForAccount(cliCtx, accountName); err != nil {
			return err
		}
	}

	accountsBackedUp := accounts[0]
	if len(accounts) > 1 {
		accountsBackedUp = strings.Join(accounts, ", ")
	}
	log.WithField("walletPath", walletDir).Infof("Successfully imported account(s) %s", accountsBackedUp)
	return nil
}

func inputImportDir(cliCtx *cli.Context) (string, error) {
	outputDir := cliCtx.String(flags.BackupPathFlag.Name)
	if outputDir == flags.DefaultValidatorDir() {
		outputDir = path.Join(outputDir)
	}
	prompt := promptui.Prompt{
		Label:    "Enter the file location of the exported wallet zip to import",
		Validate: validateDirectoryPath,
		Default:  outputDir,
	}
	outputPath, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("could not determine import directory: %v", formatPromptError(err))
	}
	return outputPath, nil
}

func unzipArchiveToTarget(archiveDir string, target string) error {
	archiveFile := filepath.Join(archiveDir, archiveFilename)
	reader, err := zip.OpenReader(archiveFile)
	if err != nil {
		return errors.Wrap(err, "could not open reader for archive")
	}

	perms := os.FileMode(0700)
	if err := os.MkdirAll(target, perms); err != nil {
		return errors.Wrap(err, "could not parent path for folder")
	}

	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		parentFolder := filepath.Dir(path)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, perms); err != nil {
				return errors.Wrap(err, "could not make path for file")
			}
			continue
		} else {
			if err := os.MkdirAll(parentFolder, perms); err != nil {
				return errors.Wrap(err, "could not make path for file")
			}
		}

		if err := copyFileFromZipToPath(file, path); err != nil {
			return err
		}
	}
	return nil
}

func copyFileFromZipToPath(file *zip.File, path string) error {
	fileReader, err := file.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err := fileReader.Close(); err != nil {
			log.WithError(err).Error("Could not close file")
		}
	}()

	targetFile, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "could not open file")
	}
	defer func() {
		if err := targetFile.Close(); err != nil {
			log.WithError(err).Error("Could not close target")
		}
	}()

	if _, err := io.Copy(targetFile, fileReader); err != nil {
		return errors.Wrap(err, "could not copy file")
	}
	return nil
}

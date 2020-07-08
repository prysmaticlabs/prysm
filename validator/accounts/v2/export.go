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

func ExportAccount(cliCtx *cli.Context) error {
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not parse wallet directory")
	}

	outputDir, err := inputZipDir(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not parse output directory")
	}

	wallet, err := OpenWallet(context.Background(), &WalletConfig{
		PasswordsDir: "", // Not needed for exporting.
		WalletDir:    walletDir,
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
		return err
	}
	return nil
}

func inputZipDir(cliCtx *cli.Context) (string, error) {
	outputDir := cliCtx.String(flags.OutputPathFlag.Name)
	if outputDir == flags.DefaultValidatorDir() {
		outputDir = path.Join(outputDir, WalletDefaultDirName[1:])
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

func selectAccounts(allAccounts []string) ([]string, error) {
	if len(allAccounts) == 1 {
		return allAccounts, nil
	}
	prompt := promptui.SelectWithAdd{
		Label: "Select accounts to backup",
		Items: append(allAccounts, "All accounts"),
	}

	_, result, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	fmt.Printf("You choose %q\n", result)
	return []string{result}, nil
}

func (w *Wallet) zipAccounts(accounts []string, targetPath string) error {
	sourcePath := w.accountsPath
	zipfile, err := os.Create(targetPath)
	if err != nil {
		return err
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

	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(sourcePath)
	}

	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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
			return err
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
			return err
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
		return err
	}

	return err
}

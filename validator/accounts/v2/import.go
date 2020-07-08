package v2

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/urfave/cli/v2"
)

func ImportAccount(cliCtx *cli.Context) error {
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if err != nil {
		log.Fatalf("Could not parse wallet directory: %v", err)
	}

	outputDir, err := inputZipDir(cliCtx)
	if err != nil {
		log.Fatalf("Could not parse output directory: %v", err)
	}

	if err := unzipFolder(outputDir, walletDir); err != nil {
		log.Fatal(err)
	}

	// Read the directory for password storage from user input.
	passwordsDirPath := inputPasswordsDirectory(cliCtx)

	wallet, err := OpenWallet(context.Background(), &WalletConfig{
		PasswordsDir: passwordsDirPath,
		WalletDir:    walletDir,
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
		// Read the new account's password from user input.
		password, err := inputPasswordForAccount(cliCtx, accountName)
		if err != nil {
			log.Fatalf("Could not read password: %v", err)
		}

		if err := wallet.writePasswordToFile(accountName, password); err != nil {
			return errors.Wrap(err, "could not write password to disk")
		}
	}
	return nil
}

func unzipFolder(archive string, target string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, file.Mode()); err != nil {
				return err
			}
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := fileReader.Close(); err != nil {
				log.WithError(err).Error("Could not close file")
			}
		}()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer func() {
			if err := targetFile.Close(); err != nil {
				log.WithError(err).Error("Could not close target")
			}
		}()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
	}

	return nil
}

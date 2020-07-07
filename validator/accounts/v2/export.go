package v2

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/prysmaticlabs/prysm/validator/flags"

	"github.com/urfave/cli/v2"
)

func ExportAccount(cliCtx *cli.Context) error {
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if err != nil {
		log.Fatalf("Could not parse wallet directory: %v", err)
	}

	outputDir, err := inputOutputDir(cliCtx)
	if err != nil {
		log.Fatalf("Could not parse output directory: %v", err)
	}

	if err := zipFolder(walletDir, outputDir); err != nil {
		return err
	}
	return nil
}

func inputOutputDir(cliCtx *cli.Context) (string, error) {
	outputDir := cliCtx.String(flags.OutputPathFlag.Name)
	if outputDir == flags.DefaultValidatorDir() {
		outputDir = path.Join(outputDir, WalletDefaultDirName)
	}
	prompt := promptui.Prompt{
		Label:    "Enter a wallet directory",
		Validate: validateDirectoryPath,
		Default:  outputDir,
	}
	outputPath, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("could not determine output directory: %v", formatPromptError(err))
	}
	return outputPath, nil
}

func zipFolder(sourcePath string, targetPath string) error {
	zipfile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

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
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})
	if err != nil {
		return err
	}

	return err
}

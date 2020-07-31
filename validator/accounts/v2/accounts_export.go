package v2

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/flags"
)

const allAccountsText = "All accounts"
const archiveFilename = "backup.zip"

// ExportAccount creates a zip archive of the selected accounts to be used in the future for importing accounts.
func ExportAccount(cliCtx *cli.Context) error {
	// TODO(#6777): Re-enable export command.
	return errors.New("this feature is unimplemented")
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

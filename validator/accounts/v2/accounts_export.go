package v2

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/urfave/cli/v2"
)

const allAccountsText = "All accounts"
const archiveFilename = "backup.zip"

// ExportAccount --
func ExportAccount(cliCtx *cli.Context) error {
	ctx := context.Background()
	wallet, err := openOrCreateWallet(cliCtx, func(cliCtx *cli.Context) (*Wallet, error) {
		return nil, errors.New(
			"no wallet found, nothing to export. Create a new wallet by running wallet-v2 create",
		)
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize wallet")
	}
	if wallet.KeymanagerKind() == v2keymanager.Remote {
		return errors.New(
			"remote wallets cannot export accounts",
		)
	}
	keymanager, err := wallet.InitializeKeymanager(ctx, true /* skip mnemonic confirm */)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	pubKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch validating public keys")
	}
	filteredPubKeys, err := selectAccountsToExport(cliCtx, pubKeys)
	if err != nil {
		return errors.Wrap(err, "could not select accounts to export")

	}
	// Ask the user for their desired password for their exported accounts.

	var keystoresToExport []*v2keymanager.Keystore
	switch wallet.KeymanagerKind() {
	case v2keymanager.Direct:
		km, ok := keymanager.(*direct.Keymanager)
		if !ok {
			return errors.New("could not assert keymanager interface to concrete type")
		}
		keystoresToExport, err = km.ExportKeystores(ctx, filteredPubKeys)
		if err != nil {
			return errors.Wrap(err, "could not export accounts for direct keymanager")
		}
	case v2keymanager.Derived:
		km, ok := keymanager.(*derived.Keymanager)
		if !ok {
			return errors.New("could not assert keymanager interface to concrete type")
		}
		//if err := km.ExportAccounts(ctx, []bls.PublicKey{}); err != nil {
		//	return errors.Wrap(err, "could not export accounts for derived keymanager")
		//}
		_ = km
		return nil
	default:
		return errors.New("keymanager kind not supported")
	}
	_ = keystoresToExport
	return nil
}

func selectAccountsToExport(cliCtx *cli.Context, pubKeys [][48]byte) ([]bls.PublicKey, error) {
	// Ask user for which accounts they wish to backup, then we
	// filter the respective public keys.
	pubKeyStrings := make([]string, len(pubKeys))
	for i, pk := range pubKeys {
		name := petnames.DeterministicName(pk[:], "-")
		pubKeyStrings[i] = fmt.Sprintf(
			"%d | %s | %#x", i, au.BrightGreen(name), au.BrightMagenta(bytesutil.Trunc(pk[:])),
		)
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
	exit := "Done selecting"
	results := make([]int, 0)
	au := aurora.NewAurora(true)
	for result != exit {
		prompt := promptui.Select{
			Label:        "Select accounts to backup",
			HideSelected: true,
			Items:        append([]string{exit, allAccountsText}, pubKeyStrings...),
			Templates:    templates,
		}

		_, result, err = prompt.Run()
		if err != nil {
			return nil, err
		}
		if result == exit {
			fmt.Printf("%s\n", au.BrightRed("Done with selections").Bold())
			break
		}
		if result == allAccountsText {
			fmt.Printf("%s\n", au.BrightRed("[Selected all accounts]").Bold())
			for i := 0; i < len(pubKeys); i++ {
				results = append(results, i)
			}
			break
		}
		idx := strings.Index(result, " |")
		accountIndexStr := result[:idx]
		accountIndex, err := strconv.Atoi(accountIndexStr)
		if err != nil {
			return nil, err
		}
		results = append(results, accountIndex)
		fmt.Printf("%s %s\n", au.BrightRed("[Selected account]").Bold(), result)
	}

	// Deduplicate the results.
	seen := make(map[int]bool)
	for i := 0; i < len(results); i++ {
		if _, ok := seen[results[i]]; !ok {
			seen[results[i]] = true
		}
	}

	// Filter the public keys for export based on user input.
	filteredPubKeys := make([]bls.PublicKey, 0)
	for selectedIndex := range seen {
		pk, err := bls.PublicKeyFromBytes(pubKeys[selectedIndex][:])
		if err != nil {
			return nil, err
		}
		filteredPubKeys = append(filteredPubKeys, pk)
	}
	return filteredPubKeys, nil
}

func (w *Wallet) zipAccounts(accounts []string, targetPath string) error {
	sourcePath := filepath.Dir(w.accountsPath)
	archivePath := filepath.Join(targetPath, archiveFilename)
	if err := os.MkdirAll(targetPath, params.BeaconIoConfig().ReadWriteExecutePermissions); err != nil {
		return errors.Wrap(err, "could not create target folder")
	}
	if fileutil.FileExists(archivePath) {
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

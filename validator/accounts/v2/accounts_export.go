package v2

import (
	"archive/zip"
	"context"
	"encoding/json"
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
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
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
	keymanager, err := wallet.InitializeKeymanager(cliCtx, true /* skip mnemonic confirm */)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	pubKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch validating public keys")
	}

	// Input the directory where they wish to export their accounts.

	// Allow the user to interactively select the accounts to export.
	filteredPubKeys, err := selectAccountsToExport(cliCtx, pubKeys)
	if err != nil {
		return errors.Wrap(err, "could not select accounts to export")
	}

	// Ask the user for their desired password for their exported accounts.
	exportsPassword, err := promptutil.InputPassword(
		cliCtx,
		flags.WalletPasswordFileFlag,
		"Enter a new password for your exported accounts",
		"Confirm new password",
		promptutil.ConfirmPass,
		promptutil.ValidatePasswordInput,
	)
	if err != nil {
		return errors.Wrap(err, "could not determine password for exported accounts")
	}

	var keystoresToExport []*v2keymanager.Keystore
	switch wallet.KeymanagerKind() {
	case v2keymanager.Direct:
		km, ok := keymanager.(*direct.Keymanager)
		if !ok {
			return errors.New("could not assert keymanager interface to concrete type")
		}
		keystoresToExport, err = km.ExportKeystores(ctx, filteredPubKeys, exportsPassword)
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
	return writeKeystoresToOutputDir(keystoresToExport, "/Users/zypherx/Desktop/mybackup")
}

// Ask user for which accounts they wish to backup via an interactive prompt.
func selectAccountsToExport(cliCtx *cli.Context, pubKeys [][48]byte) ([]bls.PublicKey, error) {
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

func writeKeystoresToOutputDir(keystoresToExport []*v2keymanager.Keystore, outputDir string) error {
	if len(keystoresToExport) == 0 {
		return errors.New("nothing to export")
	}
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return err
	}
	// Marshal and zip all keystore files together and write the zip file
	// to the specified output directory.
	archivePath := filepath.Join(outputDir, archiveFilename)
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
	// Using this zip file, we create a new zip writer which we write
	// files to directly from our marshaled keystores.
	writer := zip.NewWriter(zipfile)
	for i, k := range keystoresToExport {
		encodedFile, err := json.MarshalIndent(k, "", "\t")
		if err != nil {
			return err
		}
		f, err := writer.Create(fmt.Sprintf("keystore-%d.json", i))
		if err != nil {
			return err
		}
		_, err = f.Write(encodedFile)
		if err != nil {
			return err
		}
	}
	err = writer.Close()
	if err != nil {
		return err
	}
	return nil
}

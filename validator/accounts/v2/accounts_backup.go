package v2

import (
	"archive/zip"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/urfave/cli/v2"
)

const (
	allAccountsText  = "All accounts"
	archiveFilename  = "backup.zip"
	backupPromptText = "Enter the directory where your backup.zip file will be written to"
)

// BackupAccounts allows users to select validator accounts from their wallet
// and export them as a backup.zip file containing the keys as EIP-2335 compliant
// keystore.json files, which are compatible with importing in other eth2 clients.
func BackupAccounts(cliCtx *cli.Context) error {
	ctx := context.Background()
	wallet, err := openOrCreateWallet(cliCtx, func(cliCtx *cli.Context) (*Wallet, error) {
		return nil, errors.New(
			"no wallet found, nothing to backup. Create a new wallet by running wallet-v2 create",
		)
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize wallet")
	}
	if wallet.KeymanagerKind() == v2keymanager.Remote {
		return errors.New(
			"remote wallets cannot backup accounts",
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

	// Input the directory where they wish to backup their accounts.
	backupDir, err := inputDirectory(cliCtx, backupPromptText, flags.BackupDirFlag)
	if err != nil {
		return errors.Wrap(err, "could not parse keys directory")
	}

	// Allow the user to interactively select the accounts to backup or optionally
	// provide them via cli flags.
	filteredPubKeys, err := determinePublicKeysForBackup(cliCtx, pubKeys)
	if err != nil {
		return errors.Wrap(err, "could not filter public keys for backup")
	}

	// Ask the user for their desired password for their backed up accounts.
	backupsPassword, err := promptutil.InputPassword(
		cliCtx,
		flags.BackupsPasswordFile,
		"Enter a new password for your backed up accounts",
		"Confirm new password",
		promptutil.ConfirmPass,
		promptutil.ValidatePasswordInput,
	)
	if err != nil {
		return errors.Wrap(err, "could not determine password for backed up accounts")
	}

	var keystoresToBackup []*v2keymanager.Keystore
	switch wallet.KeymanagerKind() {
	case v2keymanager.Direct:
		km, ok := keymanager.(*direct.Keymanager)
		if !ok {
			return errors.New("could not assert keymanager interface to concrete type")
		}
		keystoresToBackup, err = km.ExtractKeystores(ctx, filteredPubKeys, backupsPassword)
		if err != nil {
			return errors.Wrap(err, "could not backup accounts for direct keymanager")
		}
	case v2keymanager.Derived:
		km, ok := keymanager.(*derived.Keymanager)
		if !ok {
			return errors.New("could not assert keymanager interface to concrete type")
		}
		_ = km
		return nil
	default:
		return errors.New("keymanager kind not supported")
	}
	return zipKeystoresToOutputDir(keystoresToBackup, backupDir)
}

func determinePublicKeysForBackup(cliCtx *cli.Context, validatingPublicKeys [][48]byte) ([]bls.PublicKey, error) {
	var filteredPubKeys []bls.PublicKey
	if cliCtx.IsSet(flags.BackupForPublicKeysFlag.Name) {
		pubKeyStrings := strings.Split(cliCtx.String(flags.BackupForPublicKeysFlag.Name), ",")
		if len(pubKeyStrings) == 0 {
			return nil, fmt.Errorf(
				"could not parse %s. It must be a string of comma-separated hex strings",
				flags.BackupForPublicKeysFlag.Name,
			)
		}
		for _, str := range pubKeyStrings {
			pkString := str
			if strings.Contains(pkString, "0x") {
				pkString = pkString[2:]
			}
			pubKeyBytes, err := hex.DecodeString(pkString)
			if err != nil {
				return nil, errors.Wrapf(err, "could not decode string %s as hex", pkString)
			}
			blsPublicKey, err := bls.PublicKeyFromBytes(pubKeyBytes)
			if err != nil {
				return nil, errors.Wrapf(err, "%#x is not a valid BLS public key", pubKeyBytes)
			}
			filteredPubKeys = append(filteredPubKeys, blsPublicKey)
		}
		return filteredPubKeys, nil
	}
	return selectAccounts(cliCtx, validatingPublicKeys)
}

// Ask user to select accounts via an interactive prompt.
func selectAccounts(cliCtx *cli.Context, pubKeys [][48]byte) ([]bls.PublicKey, error) {
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

	// Filter the public keys for backup based on user input.
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

// Zips a list of keystore into respective EIP-2335 keystore.json files and
// writes their zipped format into the specified output directory.
func zipKeystoresToOutputDir(keystoresToBackup []*v2keymanager.Keystore, outputDir string) error {
	if len(keystoresToBackup) == 0 {
		return errors.New("nothing to backup")
	}
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return errors.Wrapf(err, "could not create directory at path: %s", outputDir)
	}
	// Marshal and zip all keystore files together and write the zip file
	// to the specified output directory.
	archivePath := filepath.Join(outputDir, archiveFilename)
	if fileutil.FileExists(archivePath) {
		return errors.Errorf("Zip file already exists in directory: %s", archivePath)
	}
	// We create a new file to store our backup.zip.
	zipfile, err := os.Create(archivePath)
	if err != nil {
		return errors.Wrapf(err, "could not create zip file with path: %s", archivePath)
	}
	defer func() {
		if err := zipfile.Close(); err != nil {
			log.WithError(err).Error("Could not close zipfile")
		}
	}()
	// Using this zip file, we create a new zip writer which we write
	// files to directly from our marshaled keystores.
	writer := zip.NewWriter(zipfile)
	for i, k := range keystoresToBackup {
		encodedFile, err := json.MarshalIndent(k, "", "\t")
		if err != nil {
			return errors.Wrap(err, "could not marshal keystore to JSON file")
		}
		f, err := writer.Create(fmt.Sprintf("keystore-%d.json", i))
		if err != nil {
			return errors.Wrap(err, "could not write keystore file to zip")
		}
		if _, err = f.Write(encodedFile); err != nil {
			return errors.Wrap(err, "could not write keystore file contents")
		}
	}
	// We close the zip writer when done.
	if err := writer.Close(); err != nil {
		return errors.Wrap(err, "could not close zip file after writing")
	}
	log.WithField(
		"backup-path", archivePath,
	).Infof("Successfully backed up %d accounts", len(keystoresToBackup))
	return nil
}

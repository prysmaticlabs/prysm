package accounts

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/io/prompt"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/petnames"
	"github.com/prysmaticlabs/prysm/validator/accounts/userprompt"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/urfave/cli/v2"
)

var (
	au = aurora.NewAurora(true)
)

const (
	allAccountsText  = "All accounts"
	archiveFilename  = "backup.zip"
	backupPromptText = "Enter the directory where your backup.zip file will be written to"
)

// BackupAccountsCli allows users to select validator accounts from their wallet
// and export them as a backup.zip file containing the keys as EIP-2335 compliant
// keystore.json files, which are compatible with importing in other Ethereum consensus clients.
func BackupAccountsCli(cliCtx *cli.Context) error {
	w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		return nil, wallet.ErrNoWalletFound
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize wallet")
	}
	// TODO(#9883) - Remove this when we have a better way to handle this.
	if w.KeymanagerKind() == keymanager.Remote || w.KeymanagerKind() == keymanager.Web3Signer {
		return errors.New(
			"remote and web3signer wallets cannot backup accounts",
		)
	}
	km, err := w.InitializeKeymanager(cliCtx.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	if err != nil {
		return errors.Wrap(err, ErrCouldNotInitializeKeymanager)
	}
	pubKeys, err := km.FetchValidatingPublicKeys(cliCtx.Context)
	if err != nil {
		return errors.Wrap(err, "could not fetch validating public keys")
	}

	// Input the directory where they wish to backup their accounts.
	backupDir, err := userprompt.InputDirectory(cliCtx, backupPromptText, flags.BackupDirFlag)
	if err != nil {
		return errors.Wrap(err, "could not parse keys directory")
	}

	// Allow the user to interactively select the accounts to backup or optionally
	// provide them via cli flags as a string of comma-separated, hex strings.
	filteredPubKeys, err := filterPublicKeysFromUserInput(
		cliCtx,
		flags.BackupPublicKeysFlag,
		pubKeys,
		userprompt.SelectAccountsBackupPromptText,
	)
	if err != nil {
		return errors.Wrap(err, "could not filter public keys for backup")
	}

	// Ask the user for their desired password for their backed up accounts.
	backupsPassword, err := prompt.InputPassword(
		cliCtx,
		flags.BackupPasswordFile,
		"Enter a new password for your backed up accounts",
		"Confirm new password",
		true,
		prompt.ValidatePasswordInput,
	)
	if err != nil {
		return errors.Wrap(err, "could not determine password for backed up accounts")
	}

	keystoresToBackup, err := km.ExtractKeystores(cliCtx.Context, filteredPubKeys, backupsPassword)
	if err != nil {
		return errors.Wrap(err, "could not extract keys from keymanager")
	}
	return zipKeystoresToOutputDir(keystoresToBackup, backupDir)
}

// Ask user to select accounts via an interactive userprompt.
func selectAccounts(selectionPrompt string, pubKeys [][fieldparams.BLSPubkeyLength]byte) (filteredPubKeys []bls.PublicKey, err error) {
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
	exit := "Done selecting"
	results := make([]int, 0)
	au := aurora.NewAurora(true)
	for result != exit {
		p := promptui.Select{
			Label:        selectionPrompt,
			HideSelected: true,
			Items:        append([]string{exit, allAccountsText}, pubKeyStrings...),
			Templates:    templates,
		}

		_, result, err = p.Run()
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

	// Filter the public keys based on user input.
	filteredPubKeys = make([]bls.PublicKey, 0)
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
func zipKeystoresToOutputDir(keystoresToBackup []*keymanager.Keystore, outputDir string) error {
	if len(keystoresToBackup) == 0 {
		return errors.New("nothing to backup")
	}
	if err := file.MkdirAll(outputDir); err != nil {
		return errors.Wrapf(err, "could not create directory at path: %s", outputDir)
	}
	// Marshal and zip all keystore files together and write the zip file
	// to the specified output directory.
	archivePath := filepath.Join(outputDir, archiveFilename)
	if file.FileExists(archivePath) {
		return errors.Errorf("Zip file already exists in directory: %s", archivePath)
	}
	// We create a new file to store our backup.zip.
	zipfile, err := os.Create(filepath.Clean(archivePath))
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
	defer func() {
		// We close the zip writer when done.
		if err := writer.Close(); err != nil {
			log.WithError(err).Error("Could not close zip file after writing")
		}
	}()
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
	log.WithField(
		"backup-path", archivePath,
	).Infof("Successfully backed up %d accounts", len(keystoresToBackup))
	return nil
}

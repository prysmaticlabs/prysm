package userprompt

import (
	"fmt"
	"os"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/v3/io/file"
	"github.com/prysmaticlabs/prysm/v3/io/prompt"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote"
	"github.com/urfave/cli/v2"
)

const (
	// ImportKeysDirPromptText for the import keys cli function.
	ImportKeysDirPromptText = "Enter the directory or filepath where your keystores to import are located"
	// DataDirDirPromptText for the validator database directory.
	DataDirDirPromptText = "Enter the directory of the validator database you would like to use"
	// SlashingProtectionJSONPromptText for the EIP-3076 slashing protection JSON userprompt.
	SlashingProtectionJSONPromptText = "Enter the the filepath of your EIP-3076 Slashing Protection JSON from your previously used validator client"
	// WalletDirPromptText for the wallet.
	WalletDirPromptText = "Enter a wallet directory"
	// SelectAccountsDeletePromptText --
	SelectAccountsDeletePromptText = "Select the account(s) you would like to delete"
	// SelectAccountsBackupPromptText --
	SelectAccountsBackupPromptText = "Select the account(s) you wish to backup"
	// SelectAccountsVoluntaryExitPromptText --
	SelectAccountsVoluntaryExitPromptText = "Select the account(s) on which you wish to perform a voluntary exit"
)

var au = aurora.NewAurora(true)

// InputDirectory from the cli.
func InputDirectory(cliCtx *cli.Context, promptText string, flag *cli.StringFlag) (string, error) {
	directory := cliCtx.String(flag.Name)
	if cliCtx.IsSet(flag.Name) {
		return file.ExpandPath(directory)
	}
	// Append and log the appropriate directory name depending on the flag used.
	if flag.Name == flags.WalletDirFlag.Name {
		ok, err := file.HasDir(directory)
		if err != nil {
			return "", errors.Wrapf(err, "could not check if wallet dir %s exists", directory)
		}
		if ok {
			log.Infof("%s %s", au.BrightMagenta("(wallet path)"), directory)
			return directory, nil
		}
	}

	inputtedDir, err := prompt.DefaultPrompt(au.Bold(promptText).String(), directory)
	if err != nil {
		return "", err
	}
	if inputtedDir == directory {
		return directory, nil
	}
	return file.ExpandPath(inputtedDir)
}

// InputRemoteKeymanagerConfig via the cli.
func InputRemoteKeymanagerConfig(cliCtx *cli.Context) (*remote.KeymanagerOpts, error) {
	addr := cliCtx.String(flags.GrpcRemoteAddressFlag.Name)
	requireTls := !cliCtx.Bool(flags.DisableRemoteSignerTlsFlag.Name)
	crt := cliCtx.String(flags.RemoteSignerCertPathFlag.Name)
	key := cliCtx.String(flags.RemoteSignerKeyPathFlag.Name)
	ca := cliCtx.String(flags.RemoteSignerCACertPathFlag.Name)
	log.Info("Input desired configuration")
	var err error
	if addr == "" {
		addr, err = prompt.ValidatePrompt(
			os.Stdin,
			"Remote gRPC address (such as host.example.com:4000)",
			prompt.NotEmpty)
		if err != nil {
			return nil, err
		}
	}
	if requireTls && crt == "" {
		crt, err = prompt.ValidatePrompt(
			os.Stdin,
			"Path to TLS crt (such as /path/to/client.crt)",
			validateCertPath)
		if err != nil {
			return nil, err
		}
	}
	if requireTls && key == "" {
		key, err = prompt.ValidatePrompt(
			os.Stdin,
			"Path to TLS key (such as /path/to/client.key)",
			validateCertPath)
		if err != nil {
			return nil, err
		}
	}
	if requireTls && ca == "" {
		ca, err = prompt.ValidatePrompt(
			os.Stdin,
			"Path to certificate authority (CA) crt (such as /path/to/ca.crt)",
			validateCertPath)
		if err != nil {
			return nil, err
		}
	}

	crtPath, keyPath, caPath := "", "", ""
	if crt != "" {
		crtPath, err = file.ExpandPath(strings.TrimRight(crt, "\r\n"))
		if err != nil {
			return nil, errors.Wrapf(err, "could not determine absolute path for %s", crt)
		}
	}
	if key != "" {
		keyPath, err = file.ExpandPath(strings.TrimRight(key, "\r\n"))
		if err != nil {
			return nil, errors.Wrapf(err, "could not determine absolute path for %s", crt)
		}
	}
	if ca != "" {
		caPath, err = file.ExpandPath(strings.TrimRight(ca, "\r\n"))
		if err != nil {
			return nil, errors.Wrapf(err, "could not determine absolute path for %s", crt)
		}
	}

	newCfg := &remote.KeymanagerOpts{
		RemoteCertificate: &remote.CertificateConfig{
			RequireTls:     requireTls,
			ClientCertPath: crtPath,
			ClientKeyPath:  keyPath,
			CACertPath:     caPath,
		},
		RemoteAddr: addr,
	}
	fmt.Printf("%s\n", newCfg)
	return newCfg, nil
}

func validateCertPath(input string) error {
	if input == "" {
		return errors.New("crt path cannot be empty")
	}
	if !prompt.IsValidUnicode(input) {
		return errors.New("not valid unicode")
	}
	if !file.FileExists(input) {
		return fmt.Errorf("no crt found at path: %s", input)
	}
	return nil
}

// FormatPromptError for the user.
func FormatPromptError(err error) error {
	switch err {
	case promptui.ErrAbort:
		return errors.New("wallet creation aborted, closing")
	case promptui.ErrInterrupt:
		return errors.New("keyboard interrupt, closing")
	case promptui.ErrEOF:
		return errors.New("no input received, closing")
	default:
		return err
	}
}

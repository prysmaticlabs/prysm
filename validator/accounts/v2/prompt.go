package v2

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	"github.com/urfave/cli/v2"
)

const (
	importKeysDirPromptText      = "Enter the directory or filepath where your keystores to import are located"
	exportDirPromptText          = "Enter a file location to write the exported account(s) to"
	walletDirPromptText          = "Enter a wallet directory"
	passwordsDirPromptText       = "Directory where your account passwords are"
	newWalletPasswordPromptText  = "New wallet password"
	confirmPasswordPromptText    = "Confirm password"
	walletPasswordPromptText     = "Wallet password"
	newAccountPasswordPromptText = "New account password"
	passwordForAccountPromptText = "Enter password for account with public key %#x"
)

type passwordConfirm int

const (
	// An enum to indicate to the prompt that confirming the password is not needed.
	noConfirmPass passwordConfirm = iota
	// An enum to indicate to the prompt to confirm the password entered.
	confirmPass
)

var au = aurora.NewAurora(true)

func inputDirectory(cliCtx *cli.Context, promptText string, flag *cli.StringFlag) (string, error) {
	directory := cliCtx.String(flag.Name)
	if cliCtx.IsSet(flag.Name) {
		return expandPath(directory)
	}
	// Append and log the appropriate directory name depending on the flag used.
	if flag.Name == flags.WalletDirFlag.Name {
		ok, err := hasDir(directory)
		if err != nil {
			return "", errors.Wrapf(err, "could not check if wallet dir %s exists", directory)
		}
		if ok {
			log.Infof("%s %s", au.BrightMagenta("(wallet path)"), directory)
			return directory, nil
		}
	} else if flag.Name == flags.WalletPasswordsDirFlag.Name {
		ok, err := hasDir(directory)
		if err != nil {
			return "", errors.Wrapf(err, "could not check if passwords dir %s exists", directory)
		}
		if ok {
			log.Infof("%s %s", au.BrightMagenta("(account passwords path)"), directory)
			return directory, nil
		}
	}

	inputtedDir, err := promptutil.DefaultPrompt(au.Bold(promptText).String(), directory)
	if err != nil {
		return "", err
	}
	if inputtedDir == directory {
		return directory, nil
	}
	return expandPath(inputtedDir)
}

func inputPassword(
	cliCtx *cli.Context,
	passwordFileFlag *cli.StringFlag,
	promptText string,
	confirmPassword passwordConfirm,
	passwordValidator func(input string) error,
) (string, error) {
	if cliCtx.IsSet(passwordFileFlag.Name) {
		passwordFilePathInput := cliCtx.String(passwordFileFlag.Name)
		passwordFilePath, err := expandPath(passwordFilePathInput)
		if err != nil {
			return "", errors.Wrap(err, "could not determine absolute path of password file")
		}
		data, err := ioutil.ReadFile(passwordFilePath)
		if err != nil {
			return "", errors.Wrap(err, "could not read password file")
		}
		enteredPassword := strings.TrimRight(string(data), "\r\n")
		if err := passwordValidator(enteredPassword); err != nil {
			return "", errors.Wrap(err, "password did not pass validation")
		}
		return enteredPassword, nil
	}
	var hasValidPassword bool
	var walletPassword string
	var err error
	for !hasValidPassword {
		walletPassword, err = promptutil.PasswordPrompt(promptText, passwordValidator)
		if err != nil {
			return "", fmt.Errorf("could not read account password: %v", err)
		}

		if confirmPassword == confirmPass {
			passwordConfirmation, err := promptutil.PasswordPrompt(confirmPasswordPromptText, passwordValidator)
			if err != nil {
				return "", fmt.Errorf("could not read password confirmation: %v", err)
			}
			if walletPassword != passwordConfirmation {
				log.Error("Passwords do not match")
				continue
			}
			hasValidPassword = true
		} else {
			return walletPassword, nil
		}
	}
	return walletPassword, nil
}

func inputWeakPassword(cliCtx *cli.Context, passwordFileFlag *cli.StringFlag, promptText string) (string, error) {
	if cliCtx.IsSet(passwordFileFlag.Name) {
		passwordFilePathInput := cliCtx.String(passwordFileFlag.Name)
		passwordFilePath, err := expandPath(passwordFilePathInput)
		if err != nil {
			return "", errors.Wrap(err, "could not determine absolute path of password file")
		}
		data, err := ioutil.ReadFile(passwordFilePath)
		if err != nil {
			return "", errors.Wrap(err, "could not read password file")
		}
		return strings.TrimRight(string(data), "\r\n"), nil
	}
	walletPassword, err := promptutil.PasswordPrompt(promptText, promptutil.NotEmpty)
	if err != nil {
		return "", fmt.Errorf("could not read account password: %v", err)
	}
	return walletPassword, nil
}

func inputRemoteKeymanagerConfig(cliCtx *cli.Context) (*remote.Config, error) {
	addr := cliCtx.String(flags.GrpcRemoteAddressFlag.Name)
	crt := cliCtx.String(flags.RemoteSignerCertPathFlag.Name)
	key := cliCtx.String(flags.RemoteSignerKeyPathFlag.Name)
	ca := cliCtx.String(flags.RemoteSignerCACertPathFlag.Name)
	log.Info("Input desired configuration")
	var err error
	if addr == "" {
		addr, err = promptutil.ValidatePrompt("Remote gRPC address (such as host.example.com:4000)", promptutil.NotEmpty)
		if err != nil {
			return nil, err
		}
	}
	if crt == "" {
		crt, err = promptutil.ValidatePrompt("Path to TLS crt (such as /path/to/client.crt)", validateCertPath)
		if err != nil {
			return nil, err
		}
	}
	if key == "" {
		key, err = promptutil.ValidatePrompt("Path to TLS key (such as /path/to/client.key)", validateCertPath)
		if err != nil {
			return nil, err
		}
	}
	if ca == "" {
		ca, err = promptutil.ValidatePrompt("Path to certificate authority (CA) crt (such as /path/to/ca.crt)", validateCertPath)
		if err != nil {
			return nil, err
		}
	}
	crtPath, err := expandPath(strings.TrimRight(crt, "\r\n"))
	if err != nil {
		return nil, errors.Wrapf(err, "could not determine absolute path for %s", crt)
	}
	keyPath, err := expandPath(strings.TrimRight(key, "\r\n"))
	if err != nil {
		return nil, errors.Wrapf(err, "could not determine absolute path for %s", crt)
	}
	caPath, err := expandPath(strings.TrimRight(ca, "\r\n"))
	if err != nil {
		return nil, errors.Wrapf(err, "could not determine absolute path for %s", crt)
	}
	newCfg := &remote.Config{
		RemoteCertificate: &remote.CertificateConfig{
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
	if !promptutil.IsValidUnicode(input) {
		return errors.New("not valid unicode")
	}
	if !fileExists(input) {
		return fmt.Errorf("no crt found at path: %s", input)
	}
	return nil
}

func formatPromptError(err error) error {
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

// Expands a file path
// 1. replace tilde with users home dir
// 2. expands embedded environment variables
// 3. cleans the path, e.g. /a/b/../c -> /a/c
// Note, it has limitations, e.g. ~someuser/tmp will not be expanded
func expandPath(p string) (string, error) {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home := homeDir(); home != "" {
			p = home + p[1:]
		}
	}
	return filepath.Abs(path.Clean(os.ExpandEnv(p)))
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

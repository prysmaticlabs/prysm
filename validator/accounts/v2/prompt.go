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
	walletDirPromptText          = "Enter a wallet directory"
	passwordForAccountPromptText = "Enter password for account with public key %#x"
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

package v2

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	strongPasswords "github.com/nbutton23/zxcvbn-go"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	"github.com/urfave/cli/v2"
)

const (
	importDirPromptText          = "Enter the file location of the exported wallet zip to import"
	exportDirPromptText          = "Enter a file location to write the exported wallet to"
	walletDirPromptText          = "Enter a wallet directory"
	passwordsDirPromptText       = "Directory where passwords will be stored"
	newWalletPasswordPromptText  = "New wallet password"
	confirmPasswordPromptText    = "Confirm password"
	walletPasswordPromptText     = "Wallet password"
	newAccountPasswordPromptText = "New account password"
	passwordForAccountPromptText = "Enter password for account %s"
)

type passwordConfirm int

const (
	// Constants for passwords.
	minPasswordLength = 8
	// Min password score of 3 out of 5 based on the https://github.com/nbutton23/zxcvbn-go
	// library for strong-entropy password computation.
	minPasswordScore = 3
	// An enum to indicate the prompt that confirming the password is not needed.
	noConfirmPass passwordConfirm = iota
	// An enum to indicate the prompt to confirm the password entered.
	confirmPass
)

func inputDirectory(cliCtx *cli.Context, promptText string, flag *cli.StringFlag) (string, error) {
	directory := appendDirName(cliCtx.String(flag.Name), flag.Name)
	if cliCtx.IsSet(flag.Name) {
		return directory, nil
	}

	// Append and log the appropriate directory name depending on the flag used.
	if flag.Name == flags.WalletDirFlag.Name {
		ok, err := hasDir(directory)
		if err != nil {
			return "", errors.Wrapf(err, "could not check if wallet dir %s exists", directory)
		}
		if ok {
			au := aurora.NewAurora(true)
			log.Infof("%s %s", au.BrightMagenta("(wallet path)"), directory)
			return directory, nil
		}
	} else if flag.Name == flags.WalletPasswordsDirFlag.Name {
		ok, err := hasDir(directory)
		if err != nil {
			return "", errors.Wrapf(err, "could not check if passwords dir %s exists", directory)
		}
		if ok {
			au := aurora.NewAurora(true)
			log.Infof("%s %s", au.BrightMagenta("(account passwords path)"), directory)
			return directory, nil
		}
	}

	prompt := promptui.Prompt{
		Label:    promptText,
		Validate: validateDirectoryPath,
		Default:  directory,
	}
	inputtedDir, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("could not determine directory: %v", formatPromptError(err))
	}
	if inputtedDir == prompt.Default {
		return directory, nil
	}
	return appendDirName(inputtedDir, flag.Name), nil
}

func appendDirName(inputtedDir string, flagName string) string {
	switch flagName {
	case flags.WalletDirFlag.Name:
		inputtedDir = filepath.Join(inputtedDir, WalletDefaultDirName)
	case flags.WalletPasswordsDirFlag.Name:
		inputtedDir = filepath.Join(inputtedDir, PasswordsDefaultDirName)
	}
	return inputtedDir
}

func validateDirectoryPath(input string) error {
	if len(input) == 0 {
		return errors.New("directory path must not be empty")
	}
	return nil
}

func inputPassword(cliCtx *cli.Context, promptText string, confirmPassword passwordConfirm) (string, error) {
	if cliCtx.IsSet(flags.PasswordFileFlag.Name) {
		passwordFilePath := cliCtx.String(flags.PasswordFileFlag.Name)
		data, err := ioutil.ReadFile(passwordFilePath)
		if err != nil {
			return "", errors.Wrap(err, "could not read password file")
		}
		enteredPassword := string(data)
		if err := validatePasswordInput(enteredPassword); err != nil {
			return "", errors.Wrap(err, "password did not pass validation")
		}
		return strings.TrimRight(enteredPassword, "\r\n"), nil
	}

	var hasValidPassword bool
	var walletPassword string
	var err error
	for !hasValidPassword {
		prompt := promptui.Prompt{
			Label:    promptText,
			Validate: validatePasswordInput,
			Mask:     '*',
		}

		walletPassword, err = prompt.Run()
		if err != nil {
			return "", fmt.Errorf("could not read account password: %v", formatPromptError(err))
		}

		if confirmPassword == confirmPass {
			prompt = promptui.Prompt{
				Label: confirmPasswordPromptText,
				Mask:  '*',
			}
			confirmPassword, err := prompt.Run()
			if err != nil {
				return "", fmt.Errorf("could not read password confirmation: %v", formatPromptError(err))
			}
			if walletPassword != confirmPassword {
				log.Error("Passwords do not match")
				continue
			}
			hasValidPassword = true
		} else {
			return strings.TrimRight(walletPassword, "\r\n"), nil
		}
	}
	return strings.TrimRight(walletPassword, "\r\n"), nil
}

// Validate a strong password input for new accounts,
// including a min length, at least 1 number and at least
// 1 special character.
func validatePasswordInput(input string) error {
	var (
		hasMinLen  = false
		hasLetter  = false
		hasNumber  = false
		hasSpecial = false
	)
	if len(input) >= minPasswordLength {
		hasMinLen = true
	}
	for _, char := range input {
		switch {
		case !(unicode.IsLetter(char) || unicode.IsNumber(char) || unicode.IsPunct(char) || unicode.IsSymbol(char)):
			return errors.New("password must only contain alphanumeric characters, punctuation, or symbols")
		case unicode.IsLetter(char):
			hasLetter = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	if !(hasMinLen && hasLetter && hasNumber && hasSpecial) {
		return errors.New(
			"password must have more than 8 characters, at least 1 special character, and 1 number",
		)
	}
	strength := strongPasswords.PasswordStrength(input, nil)
	if strength.Score < minPasswordScore {
		return errors.New(
			"password is too easy to guess, try a stronger password",
		)
	}
	return nil
}

func inputRemoteKeymanagerConfig(cliCtx *cli.Context) (*remote.Config, error) {
	addr := cliCtx.String(flags.GrpcRemoteAddressFlag.Name)
	crt := cliCtx.String(flags.RemoteSignerCertPathFlag.Name)
	key := cliCtx.String(flags.RemoteSignerKeyPathFlag.Name)
	ca := cliCtx.String(flags.RemoteSignerCACertPathFlag.Name)

	if addr != "" && crt != "" && key != "" && ca != "" {
		if !(isValidUnicode(addr) && isValidUnicode(crt) && isValidUnicode(key) && isValidUnicode(ca)) {
			return nil, errors.New("flag inputs contain non-unicode characters")
		}
		newCfg := &remote.Config{
			RemoteCertificate: &remote.CertificateConfig{
				ClientCertPath: strings.TrimRight(crt, "\r\n"),
				ClientKeyPath:  strings.TrimRight(key, "\r\n"),
				CACertPath:     strings.TrimRight(ca, "\r\n"),
			},
			RemoteAddr: strings.TrimRight(addr, "\r\n"),
		}
		log.Infof("New configuration")
		fmt.Printf("%s\n", newCfg)
		return newCfg, nil
	}
	log.Infof("Input desired configuration")
	prompt := promptui.Prompt{
		Label: "Remote gRPC address (such as host.example.com:4000)",
		Validate: func(input string) error {
			if input == "" {
				return errors.New("remote host address cannot be empty")
			}
			if !isValidUnicode(input) {
				return errors.New("not valid unicode")
			}
			return nil
		},
	}
	remoteAddr, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	prompt = promptui.Prompt{
		Label:    "Path to TLS crt (such as /path/to/client.crt)",
		Validate: validateCertPath,
	}
	clientCrtPath, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	prompt = promptui.Prompt{
		Label:    "Path to TLS key (such as /path/to/client.key)",
		Validate: validateCertPath,
	}
	clientKeyPath, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	prompt = promptui.Prompt{
		Label:    "(Optional) Path to certificate authority (CA) crt (such as /path/to/ca.crt)",
		Validate: validateCACertPath,
	}
	caCrtPath, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	newCfg := &remote.Config{
		RemoteCertificate: &remote.CertificateConfig{
			ClientCertPath: strings.TrimRight(clientCrtPath, "\r\n"),
			ClientKeyPath:  strings.TrimRight(clientKeyPath, "\r\n"),
			CACertPath:     strings.TrimRight(caCrtPath, "\r\n"),
		},
		RemoteAddr: strings.TrimRight(remoteAddr, "\r\n"),
	}
	fmt.Printf("%s\n", newCfg)
	return newCfg, nil
}

func validateCertPath(input string) error {
	if input == "" {
		return errors.New("crt path cannot be empty")
	}
	if !isValidUnicode(input) {
		return errors.New("not valid unicode")
	}
	if !fileExists(input) {
		return fmt.Errorf("no crt found at path: %s", input)
	}
	return nil
}

func validateCACertPath(input string) error {
	if input != "" && !fileExists(input) {
		return fmt.Errorf("no crt found at path: %s", input)
	}
	if !isValidUnicode(input) {
		return errors.New("not valid unicode")
	}
	return nil
}

func validateDirectoryPath(input string) error {
	if len(input) == 0 {
		return errors.New("directory path must not be empty")
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

// Checks if an input string is a valid unicode string comprised of only
// letters, numbers, punctuation, or symbols.
func isValidUnicode(input string) bool {
	for _, char := range input {
		if !(unicode.IsLetter(char) ||
			unicode.IsNumber(char) ||
			unicode.IsPunct(char) ||
			unicode.IsSymbol(char)) {
			log.Info(char)
			return false
		}
	}
	return true
}

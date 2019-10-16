package node

import (
	"bufio"
	"os"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
)

func keysParser(ctx *cli.Context) (map[string]*keystore.Key, error) {
	// Unsafe start from plain text keys.
	if unencryptedKeys := ctx.String(flags.UnencryptedKeysFlag.Name); unencryptedKeys != "" {
		return accounts.LoadUnencryptedKeys(unencryptedKeys)
	}
	// Interop start from generated keys.
	if numValidatorKeys := ctx.GlobalUint64(flags.InteropNumValidators.Name); numValidatorKeys > 0 {
		return accounts.InteropValidatorKeys(ctx.GlobalUint64(flags.InteropStartIndex.Name), numValidatorKeys)
	}

	// Normal production key start.
	keystoreDirectory := ctx.String(flags.KeystorePathFlag.Name)
	keystorePassword := ctx.String(flags.PasswordFlag.Name)

	exists, err := accounts.Exists(keystoreDirectory)
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		// If an account does not exist, we create a new one and start the node.
		keystoreDirectory, keystorePassword, err = CreateValidatorAccount(ctx)
		if err != nil {
			log.Fatalf("Could not create validator account: %v", err)
		}
	} else {
		if keystorePassword == "" {
			log.Info("Enter your validator account password:")
			bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				log.Fatalf("Could not read account password: %v", err)
			}
			text := string(bytePassword)
			keystorePassword = strings.Replace(text, "\n", "", -1)
		}

		if err := accounts.VerifyAccountNotExists(keystoreDirectory, keystorePassword); err == nil {
			log.Info("No account found, creating new validator account...")
		}
	}
	return accounts.DecryptKeysFromKeystore(keystoreDirectory, keystorePassword)
}

// CreateValidatorAccount creates a validator account from the given cli context.
func CreateValidatorAccount(ctx *cli.Context) (string, string, error) {
	keystoreDirectory := ctx.String(flags.KeystorePathFlag.Name)
	keystorePassword := ctx.String(flags.PasswordFlag.Name)
	if keystorePassword == "" {
		reader := bufio.NewReader(os.Stdin)
		logrus.Info("Create a new validator account for eth2")
		log.Info("Enter a password:")
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("Could not read account password: %v", err)
		}
		text := string(bytePassword)
		keystorePassword = strings.Replace(text, "\n", "", -1)
		log.Infof("Keystore path to save your private keys (leave blank for default %s):", keystoreDirectory)
		text, err = reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		text = strings.Replace(text, "\n", "", -1)
		if text != "" {
			keystoreDirectory = text
		}
	}

	if err := accounts.NewValidatorAccount(keystoreDirectory, keystorePassword); err != nil {
		return "", "", errors.Wrapf(err, "could not initialize validator account")
	}
	return keystoreDirectory, keystorePassword, nil
}

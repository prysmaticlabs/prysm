package direct

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/k0kubun/go-ansi"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

// ImportKeystores into the direct keymanager from an external source.
func (dr *Keymanager) ImportKeystores(cliCtx *cli.Context, keystores []*v2keymanager.Keystore) error {
	decryptor := keystorev4.New()
	privKeys := make([][]byte, len(keystores))
	pubKeys := make([][]byte, len(keystores))
	bar := initializeProgressBar(len(keystores), "Importing accounts...")
	var password string
	var err error
	if cliCtx.IsSet(flags.AccountPasswordFileFlag.Name) {
		passwordFilePath := cliCtx.String(flags.AccountPasswordFileFlag.Name)
		data, err := ioutil.ReadFile(passwordFilePath)
		if err != nil {
			return err
		}
		password = string(data)
	} else {
		password, err = promptutil.PasswordPrompt(
			"Enter the password for your imported accounts", promptutil.NotEmpty,
		)
		if err != nil {
			return fmt.Errorf("could not read account password: %v", err)
		}
	}
	fmt.Println("Importing accounts, this may take a while...")
	for i := 0; i < len(keystores); i++ {
		privKeyBytes, pubKeyBytes, err := dr.attemptDecryptKeystore(decryptor, keystores[i], password)
		if err != nil {
			return err
		}
		privKeys[i] = privKeyBytes
		pubKeys[i] = pubKeyBytes
		if err := bar.Add(1); err != nil {
			return errors.Wrap(err, "could not add to progress bar")
		}
		fmt.Printf(
			"Successfully imported account with public key %#x\n",
			aurora.BrightMagenta(bytesutil.Trunc(pubKeyBytes)),
		)
	}
	// Write the accounts to disk into a single keystore.
	ctx := context.Background()
	accountsKeystore, err := dr.createAccountsKeystore(ctx, privKeys, pubKeys)
	if err != nil {
		return err
	}
	encodedAccounts, err := json.MarshalIndent(accountsKeystore, "", "\t")
	if err != nil {
		return err
	}
	fileName := fmt.Sprintf(accountsKeystoreFileNameFormat, roughtime.Now().Unix())
	return dr.wallet.WriteFileAtPath(ctx, AccountsPath, fileName, encodedAccounts)
}

// Retrieves the private key and public key from an EIP-2335 keystore file
// by decrypting using a specified password. If the password fails,
// it prompts the user for the correct password until it confirms.
func (dr *Keymanager) attemptDecryptKeystore(
	enc *keystorev4.Encryptor, keystore *v2keymanager.Keystore, password string,
) ([]byte, []byte, error) {
	// Attempt to decrypt the keystore with the specifies password.
	var privKeyBytes []byte
	var err error
	privKeyBytes, err = enc.Decrypt(keystore.Crypto, password)
	if err != nil && strings.Contains(err.Error(), "invalid checksum") {
		// If the password fails for an individual account, we ask the user to input
		// that individual account's password until it succeeds.
		privKeyBytes, err = dr.askUntilPasswordConfirms(enc, keystore)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not confirm password via prompt")
		}
	} else if err != nil {
		return nil, nil, errors.Wrap(err, "could not decrypt keystore")
	}
	pubKeyBytes, err := hex.DecodeString(keystore.Pubkey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not decode pubkey from keystore")
	}
	return privKeyBytes, pubKeyBytes, nil
}

func initializeProgressBar(numItems int, msg string) *progressbar.ProgressBar {
	return progressbar.NewOptions(
		numItems,
		progressbar.OptionFullWidth(),
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() { fmt.Println() }),
		progressbar.OptionSetDescription(msg),
	)
}

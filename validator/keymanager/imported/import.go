package imported

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/k0kubun/go-ansi"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/schollz/progressbar/v3"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

// ImportKeystores into the imported keymanager from an external source.
func (dr *Keymanager) ImportKeystores(
	ctx context.Context,
	keystores []*keymanager.Keystore,
	importsPassword string,
) error {
	decryptor := keystorev4.New()
	bar := initializeProgressBar(len(keystores), "Importing accounts...")
	keys := map[string]string{}
	var err error
	for i := 0; i < len(keystores); i++ {
		var privKeyBytes []byte
		var pubKeyBytes []byte
		privKeyBytes, pubKeyBytes, importsPassword, err = dr.attemptDecryptKeystore(decryptor, keystores[i], importsPassword)
		if err != nil {
			return err
		}
		// if key exists prior to being added then output log that duplicate key was found
		if _, ok := keys[string(pubKeyBytes)]; ok {
			log.Warnf("Duplicate key in import folder will be ignored: %#x", pubKeyBytes)
		}
		keys[string(pubKeyBytes)] = string(privKeyBytes)
		if err := bar.Add(1); err != nil {
			return errors.Wrap(err, "could not add to progress bar")
		}
	}
	privKeys := make([][]byte, 0)
	pubKeys := make([][]byte, 0)
	for pubKey, privKey := range keys {
		pubKeys = append(pubKeys, []byte(pubKey))
		privKeys = append(privKeys, []byte(privKey))
	}

	// Write the accounts to disk into a single keystore.
	accountsKeystore, err := dr.createAccountsKeystore(ctx, privKeys, pubKeys)
	if err != nil {
		return err
	}
	encodedAccounts, err := json.MarshalIndent(accountsKeystore, "", "\t")
	if err != nil {
		return err
	}
	return dr.wallet.WriteFileAtPath(ctx, AccountsPath, accountsKeystoreFileName, encodedAccounts)
}

// ImportKeypairs directly into the keymanager.
func (dr *Keymanager) ImportKeypairs(ctx context.Context, privKeys, pubKeys [][]byte) error {
	// Write the accounts to disk into a single keystore.
	accountsKeystore, err := dr.createAccountsKeystore(ctx, privKeys, pubKeys)
	if err != nil {
		return errors.Wrap(err, "could not import account keypairs")
	}
	encodedAccounts, err := json.MarshalIndent(accountsKeystore, "", "\t")
	if err != nil {
		return errors.Wrap(err, "could not marshal accounts keystore into JSON")
	}
	return dr.wallet.WriteFileAtPath(ctx, AccountsPath, accountsKeystoreFileName, encodedAccounts)
}

// Retrieves the private key and public key from an EIP-2335 keystore file
// by decrypting using a specified password. If the password fails,
// it prompts the user for the correct password until it confirms.
func (dr *Keymanager) attemptDecryptKeystore(
	enc *keystorev4.Encryptor, keystore *keymanager.Keystore, password string,
) ([]byte, []byte, string, error) {
	// Attempt to decrypt the keystore with the specifies password.
	var privKeyBytes []byte
	var err error
	privKeyBytes, err = enc.Decrypt(keystore.Crypto, password)
	doesNotDecrypt := err != nil && strings.Contains(err.Error(), "invalid checksum")
	for doesNotDecrypt {
		password, err = promptutil.PasswordPrompt(
			"Password incorrect for keystore, input correct password", promptutil.NotEmpty,
		)
		if err != nil {
			return nil, nil, "", fmt.Errorf("could not read keystore password: %w", err)
		}
		privKeyBytes, err = enc.Decrypt(keystore.Crypto, password)
		doesNotDecrypt = err != nil && strings.Contains(err.Error(), "invalid checksum")
		if err != nil && !strings.Contains(err.Error(), "invalid checksum") {
			return nil, nil, "", errors.Wrap(err, "could not decrypt keystore")
		}
	}
	if err != nil && !strings.Contains(err.Error(), "invalid checksum") {
		return nil, nil, "", errors.Wrap(err, "could not decrypt keystore")
	}
	var pubKeyBytes []byte
	// Attempt to use the pubkey present in the keystore itself as a field. If unavailable,
	// then utilize the public key directly from the private key.
	if keystore.Pubkey != "" {
		pubKeyBytes, err = hex.DecodeString(keystore.Pubkey)
		if err != nil {
			return nil, nil, "", errors.Wrap(err, "could not decode pubkey from keystore")
		}
	} else {
		privKey, err := bls.SecretKeyFromBytes(privKeyBytes)
		if err != nil {
			return nil, nil, "", errors.Wrap(err, "could not initialize private key from bytes")
		}
		pubKeyBytes = privKey.PublicKey().Marshal()
	}
	return privKeyBytes, pubKeyBytes, password, nil
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

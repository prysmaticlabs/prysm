package local

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/k0kubun/go-ansi"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	ethpbservice "github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/schollz/progressbar/v3"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

// ImportKeystores into the local keymanager from an external source.
func (km *Keymanager) ImportKeystores(
	ctx context.Context,
	keystores []*keymanager.Keystore,
	passwords []string,
) ([]*ethpbservice.ImportedKeystoreStatus, error) {
	if len(passwords) == 0 {
		return nil, ErrNoPasswords
	}
	if len(passwords) != len(keystores) {
		return nil, ErrMismatchedNumPasswords
	}
	decryptor := keystorev4.New()
	bar := initializeProgressBar(len(keystores), "Importing accounts...")
	keys := map[string]string{}
	statuses := make([]*ethpbservice.ImportedKeystoreStatus, len(keystores))
	var err error

	for i := 0; i < len(keystores); i++ {
		var privKeyBytes []byte
		var pubKeyBytes []byte
		privKeyBytes, pubKeyBytes, _, err = km.attemptDecryptKeystore(decryptor, keystores[i], passwords[i])
		if err != nil {
			statuses[i] = &ethpbservice.ImportedKeystoreStatus{
				Status:  ethpbservice.ImportedKeystoreStatus_ERROR,
				Message: err.Error(),
			}
			continue
		}
		if err := bar.Add(1); err != nil {
			log.Error(err)
		}
		// if key exists prior to being added then output log that duplicate key was found
		if _, ok := keys[string(pubKeyBytes)]; ok {
			log.Warnf("Duplicate key in import will be ignored: %#x", pubKeyBytes)
			statuses[i] = &ethpbservice.ImportedKeystoreStatus{
				Status: ethpbservice.ImportedKeystoreStatus_DUPLICATE,
			}
			continue
		}
		keys[string(pubKeyBytes)] = string(privKeyBytes)
		statuses[i] = &ethpbservice.ImportedKeystoreStatus{
			Status: ethpbservice.ImportedKeystoreStatus_IMPORTED,
		}
	}
	privKeys := make([][]byte, 0)
	pubKeys := make([][]byte, 0)
	for pubKey, privKey := range keys {
		pubKeys = append(pubKeys, []byte(pubKey))
		privKeys = append(privKeys, []byte(privKey))
	}

	// Write the accounts to disk into a single keystore.
	accountsKeystore, err := km.CreateAccountsKeystore(ctx, privKeys, pubKeys)
	if err != nil {
		return nil, err
	}
	encodedAccounts, err := json.MarshalIndent(accountsKeystore, "", "\t")
	if err != nil {
		return nil, err
	}
	if err := km.wallet.WriteFileAtPath(ctx, AccountsPath, AccountsKeystoreFileName, encodedAccounts); err != nil {
		return nil, err
	}
	return statuses, nil
}

// ImportKeypairs directly into the keymanager.
func (km *Keymanager) ImportKeypairs(ctx context.Context, privKeys, pubKeys [][]byte) error {
	// Write the accounts to disk into a single keystore.
	accountsKeystore, err := km.CreateAccountsKeystore(ctx, privKeys, pubKeys)
	if err != nil {
		return errors.Wrap(err, "could not import account keypairs")
	}
	encodedAccounts, err := json.MarshalIndent(accountsKeystore, "", "\t")
	if err != nil {
		return errors.Wrap(err, "could not marshal accounts keystore into JSON")
	}
	return km.wallet.WriteFileAtPath(ctx, AccountsPath, AccountsKeystoreFileName, encodedAccounts)
}

// Retrieves the private key and public key from an EIP-2335 keystore file
// by decrypting using a specified password. If the password fails,
// it prompts the user for the correct password until it confirms.
func (_ *Keymanager) attemptDecryptKeystore(
	enc *keystorev4.Encryptor, keystore *keymanager.Keystore, password string,
) ([]byte, []byte, string, error) {
	// Attempt to decrypt the keystore with the specifies password.
	var privKeyBytes []byte
	var err error
	privKeyBytes, err = enc.Decrypt(keystore.Crypto, password)
	doesNotDecrypt := err != nil && strings.Contains(err.Error(), keymanager.IncorrectPasswordErrMsg)
	if doesNotDecrypt {
		return nil, nil, "", fmt.Errorf(
			"incorrect password for key 0x%s",
			keystore.Pubkey,
		)
	}
	if err != nil && !strings.Contains(err.Error(), keymanager.IncorrectPasswordErrMsg) {
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

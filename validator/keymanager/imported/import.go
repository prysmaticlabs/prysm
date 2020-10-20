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
	privKeys := make([][]byte, len(keystores))
	pubKeys := make([][]byte, len(keystores))
	bar := initializeProgressBar(len(keystores), "Importing accounts...")
	var err error
	var privKeyBytes []byte
	var pubKeyBytes []byte
	for i := 0; i < len(keystores); i++ {
		privKeyBytes, pubKeyBytes, importsPassword, err = dr.attemptDecryptKeystore(decryptor, keystores[i], importsPassword)
		if err != nil {
			return err
		}
		privKeys[i] = privKeyBytes
		pubKeys[i] = pubKeyBytes
		if err := bar.Add(1); err != nil {
			return errors.Wrap(err, "could not add to progress bar")
		}
	}
	foundKey := map[string]bool{}
	for i := range pubKeys {
		strKey := string(pubKeys[i])
		if foundKey[strKey] {
			return fmt.Errorf("duplicated key found: %#x", pubKeys[i])
		}
		foundKey[strKey] = true
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
	if err != nil && strings.Contains(err.Error(), "invalid checksum") {
		return nil, nil, "", fmt.Errorf("wrong password for keystore with pubkey %s", keystore.Pubkey)
	} else if err != nil {
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

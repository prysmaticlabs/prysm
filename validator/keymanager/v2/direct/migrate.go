package direct

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

func (dr *Keymanager) migrateToSingleKeystore(ctx context.Context) error {
	accountNames, err := dr.wallet.ListDirs()
	if err != nil {
		return err
	}
	for _, name := range accountNames {
		// If the user is already using the single keystore format,
		// we have no need to migrate and we exit normally.
		if strings.Contains(name, accountsPath) {
			return nil
		}
	}
	decryptor := keystorev4.New()
	privKeys := make([][]byte, len(accountNames))
	pubKeys := make([][]byte, len(accountNames))
	// Next up, we retrieve every single keystore for each
	// account and attempt to unlock.
	for i, name := range accountNames {
		password, err := dr.wallet.ReadPasswordFromDisk(ctx, name+PasswordFileSuffix)
		if err != nil {
			return errors.Wrapf(err, "could not read password for account %s", name)
		}
		encoded, err := dr.wallet.ReadFileAtPath(ctx, name, KeystoreFileName)
		if err != nil {
			return errors.Wrapf(err, "could not read keystore file for account %s", name)
		}
		keystoreFile := &v2keymanager.Keystore{}
		if err := json.Unmarshal(encoded, keystoreFile); err != nil {
			return errors.Wrapf(err, "could not decode keystore file for account %s", name)
		}
		// We extract the validator signing private key from the keystore
		// by utilizing the password and initialize a new BLS secret key from
		// its raw bytes.
		privKeyBytes, err := decryptor.Decrypt(keystoreFile.Crypto, password)
		if err != nil {
			return errors.Wrapf(err, "could not decrypt signing key for account %s", name)
		}
		publicKeyBytes, err := hex.DecodeString(keystoreFile.Pubkey)
		if err != nil {
			return err
		}
		privKeys[i] = privKeyBytes
		pubKeys[i] = publicKeyBytes
	}
	accountsKeystore, err := dr.createAccountsKeystore(ctx, privKeys, pubKeys)
	if err != nil {
		return err
	}
	encodedAccounts, err := json.MarshalIndent(accountsKeystore, "", "\t")
	if err != nil {
		return err
	}
	fileName := fmt.Sprintf(accountsKeystoreFileNameFormat, roughtime.Now().Unix())
	return dr.wallet.WriteFileAtPath(ctx, accountsPath, fileName, encodedAccounts)
}

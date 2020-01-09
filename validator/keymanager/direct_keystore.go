package keymanager

import (
	"strings"
	"syscall"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"golang.org/x/crypto/ssh/terminal"
)

// Keystore is a key manager that loads keys from a standard keystore.
type Keystore struct {
	*Direct
}

// NewKeystore creates a key manager populated with the keys from the keystore at the given path.
func NewKeystore(path string, passphrase string) (KeyManager, error) {
	exists, err := accounts.Exists(path)
	if err != nil {
		return nil, err
	}
	if !exists {
		// If an account does not exist, we create a new one and start the node.
		path, passphrase, err = accounts.CreateValidatorAccount(path, passphrase)
		if err != nil {
			return nil, err
		}
	} else {
		if passphrase == "" {
			log.Info("Enter your validator account password:")
			bytePassword, err := terminal.ReadPassword(syscall.Stdin)
			if err != nil {
				return nil, err
			}
			text := string(bytePassword)
			passphrase = strings.Replace(text, "\n", "", -1)
		}

		if err := accounts.VerifyAccountNotExists(path, passphrase); err == nil {
			log.Info("No account found, creating new validator account...")
		}
	}

	keyMap, err := accounts.DecryptKeysFromKeystore(path, passphrase)
	if err != nil {
		return nil, err
	}

	km := &Unencrypted{
		Direct: &Direct{
			publicKeys: make(map[[48]byte]*bls.PublicKey),
			secretKeys: make(map[[48]byte]*bls.SecretKey),
		},
	}
	for _, key := range keyMap {
		pubKey := bytesutil.ToBytes48(key.PublicKey.Marshal())
		km.publicKeys[pubKey] = key.PublicKey
		km.secretKeys[pubKey] = key.SecretKey
	}
	return km, nil
}

package keymanager

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/prysmaticlabs/prysm/shared/params"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"golang.org/x/crypto/ssh/terminal"
)

// Keystore is a key manager that loads keys from a standard keystore.
type Keystore struct {
	*Direct
}

type keystoreOpts struct {
	Path       string `json:"path"`
	Passphrase string `json:"passphrase"`
}

var keystoreOptsHelp = `The keystore key manager generates keys and stores them in a local encrypted store.  The options are:
  - path This is the filesystem path to where keys will be stored.  Defaults to the user's home directory if not supplied
  - passphrase This is the passphrase used to encrypt keys.  Will be asked for if not supplied
A sample set of options are:
  {
    "path":   "/home/me/keys", // Store the keys in '/home/me/keys'
    "passphrase": "secret"     // Use the passphrase 'secret' to encrypt and decrypt keys
  }`

// NewKeystore creates a key manager populated with the keys from the keystore at the given path.
func NewKeystore(input string) (KeyManager, string, error) {
	opts := &keystoreOpts{}
	err := json.Unmarshal([]byte(input), opts)
	if err != nil {
		return nil, keystoreOptsHelp, err
	}

	if strings.Contains(opts.Path, "$") || strings.Contains(opts.Path, "~") || strings.Contains(opts.Path, "%") {
		log.WithField("path", opts.Path).Warn("Keystore path contains unexpanded shell expansion characters")
	}

	if opts.Path == "" {
		opts.Path = accounts.DefaultValidatorDir()
	}
	log.WithField("keystorePath", opts.Path).Info("Checking validator keys")

	exists, err := accounts.Exists(opts.Path, true /* assertNonEmpty */)
	if err != nil {
		return nil, keystoreOptsHelp, err
	}
	if exists {
		if opts.Passphrase == "" {
			log.Info("Enter your validator account password:")
			bytePassword, err := terminal.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return nil, keystoreOptsHelp, err
			}
			text := string(bytePassword)
			opts.Passphrase = strings.Replace(text, "\n", "", -1)
		}

		if err := accounts.VerifyAccountNotExists(opts.Path, opts.Passphrase); err == nil {
			log.Info("No account found, creating new validator account...")
		}
	} else {
		return nil, "", errors.New("no validator keys found, please use validator accounts create")
	}

	keyMap, err := accounts.DecryptKeysFromKeystore(opts.Path, params.BeaconConfig().ValidatorPrivkeyFileName, opts.Passphrase)
	if err != nil {
		return nil, keystoreOptsHelp, err
	}

	km := &Unencrypted{
		Direct: &Direct{
			publicKeys: make(map[[48]byte]bls.PublicKey),
			secretKeys: make(map[[48]byte]bls.SecretKey),
		},
	}
	for _, key := range keyMap {
		pubKey := bytesutil.ToBytes48(key.PublicKey.Marshal())
		km.publicKeys[pubKey] = key.PublicKey
		km.secretKeys[pubKey] = key.SecretKey
	}
	return km, "", nil
}

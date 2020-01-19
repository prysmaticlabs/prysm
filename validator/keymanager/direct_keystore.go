package keymanager

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
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

	if opts.Path == "" {
		opts.Path = defaultValidatorDir()
	}

	exists, err := accounts.Exists(opts.Path)
	if err != nil {
		return nil, keystoreOptsHelp, err
	}
	if !exists {
		// If an account does not exist, we create a new one and start the node.
		opts.Path, opts.Passphrase, err = accounts.CreateValidatorAccount(opts.Path, opts.Passphrase)
		if err != nil {
			return nil, keystoreOptsHelp, err
		}
	} else {
		if opts.Passphrase == "" {
			log.Info("Enter your validator account password:")
			bytePassword, err := terminal.ReadPassword(syscall.Stdin)
			if err != nil {
				return nil, keystoreOptsHelp, err
			}
			text := string(bytePassword)
			opts.Passphrase = strings.Replace(text, "\n", "", -1)
		}

		if err := accounts.VerifyAccountNotExists(opts.Path, opts.Passphrase); err == nil {
			log.Info("No account found, creating new validator account...")
		}
	}

	keyMap, err := accounts.DecryptKeysFromKeystore(opts.Path, opts.Passphrase)
	if err != nil {
		return nil, keystoreOptsHelp, err
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
	return km, "", nil
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

func defaultValidatorDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Eth2Validators")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "Eth2Validators")
		} else {
			return filepath.Join(home, ".eth2validators")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

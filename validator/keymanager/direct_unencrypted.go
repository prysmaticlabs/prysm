package keymanager

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// Unencrypted is a key manager that loads keys from an unencrypted store.
type Unencrypted struct {
	*Direct
}

type unencryptedOpts struct {
	Path string `json:"path"`
}

var unencryptedOptsHelp = `The unencrypted key manager stores keys in a local unencrypted store.  The options are:
  - path This is the filesystem path to a file containing the unencrypted keys
A sample set of options are:
  {
    "path": "/home/me/keys.json" // Access the keys in '/home/me/keys.json'
  }`

// NewUnencrypted creates a keymanager from a file of unencrypted keys.
func NewUnencrypted(input string) (*Unencrypted, string, error) {
	opts := &unencryptedOpts{}
	err := json.Unmarshal([]byte(input), opts)
	if err != nil {
		return nil, unencryptedOptsHelp, err
	}

	if strings.Contains(opts.Path, "$") || strings.Contains(opts.Path, "~") || strings.Contains(opts.Path, "%") {
		log.WithField("path", opts.Path).Warn("Keystore path contains unexpanded shell expansion characters")
	}

	path, err := filepath.Abs(opts.Path)
	if err != nil {
		return nil, unencryptedOptsHelp, err
	}
	reader, err := os.Open(path)
	if err != nil {
		return nil, unencryptedOptsHelp, err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			log.WithError(err).Error("Failed to close file reader")
		}
	}()

	keyMap, err := unencryptedKeysFromReader(reader)
	if err != nil {
		return nil, unencryptedOptsHelp, err
	}
	sks := make([]bls.SecretKey, 0, len(keyMap))
	for _, key := range keyMap {
		sks = append(sks, key)
	}

	km := &Unencrypted{
		Direct: &Direct{
			publicKeys: make(map[[48]byte]bls.PublicKey),
			secretKeys: make(map[[48]byte]bls.SecretKey),
		},
	}
	for i := 0; i < len(sks); i++ {
		pk := sks[i].PublicKey()
		pubKey := bytesutil.ToBytes48(pk.Marshal())
		km.publicKeys[pubKey] = pk
		km.secretKeys[pubKey] = sks[i]
	}
	return km, "", nil
}

type unencryptedKeysContainer struct {
	Keys []*unencryptedKeys `json:"keys"`
}

type unencryptedKeys struct {
	ValidatorKey []byte `json:"validator_key"`
}

// unencryptedKeysFromReader loads the unencrypted keys from the given reader.
func unencryptedKeysFromReader(reader io.Reader) ([]bls.SecretKey, error) {
	log.Warn("Loading encrypted keys from disk. Do not do this in production!")

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var ctnr *unencryptedKeysContainer
	if err := json.Unmarshal(data, &ctnr); err != nil {
		return nil, err
	}

	res := make([]bls.SecretKey, 0, len(ctnr.Keys))
	for i := range ctnr.Keys {
		secretKey, err := bls.SecretKeyFromBytes(ctnr.Keys[i].ValidatorKey)
		if err != nil {
			return nil, err
		}
		res = append(res, secretKey)
	}
	return res, nil
}

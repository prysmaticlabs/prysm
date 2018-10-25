package keystore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pborman/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

const (
	keyHeaderKDF = "scrypt"

	// StandardScryptN is the N parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptN = 1 << 18

	// StandardScryptP is the P parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptP = 1

	// LightScryptN is the N parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptN = 1 << 12

	// LightScryptP is the P parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptP = 6

	scryptR     = 8
	scryptDKLen = 32
)

// Key is the object that stores all the user data related to their public/secret keys.
type Key struct {
	ID uuid.UUID // Version 4 "random" for unique id not derived from key data

	PublicKey *bls.PublicKey // Represents the public key of the user.

	SecretKey *bls.SecretKey // Represents the private key of the user.
}

type keyStore interface {
	// Loads and decrypts the key from disk.
	GetKey(filename string, password string) (*Key, error)
	// Writes and encrypts the key.
	StoreKey(filename string, k *Key, auth string) error
	// Joins filename with the key directory unless it is already absolute.
	JoinPath(filename string) string
}

type plainKeyJSON struct {
	PublicKey string `json:"address"`
	SecretKey string `json:"privatekey"`
	ID        string `json:"id"`
}

type encryptedKeyJSON struct {
	PublicKey string     `json:"publickey"`
	Crypto    cryptoJSON `json:"crypto"`
	ID        string     `json:"id"`
}

type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

// MarshalJSON marshalls a key struct into a JSON blob.
func (k *Key) MarshalJSON() (j []byte, err error) {
	jStruct := plainKeyJSON{
		hex.EncodeToString(k.PublicKey.BufferedPublicKey()),
		hex.EncodeToString(k.SecretKey.BufferedSecretKey()),
		k.ID.String(),
	}
	j, err = json.Marshal(jStruct)
	return j, err
}

// UnmarshalJSON unmarshals a blob into a key struct.
func (k *Key) UnmarshalJSON(j []byte) (err error) {
	keyJSON := new(plainKeyJSON)
	err = json.Unmarshal(j, &keyJSON)
	if err != nil {
		return err
	}

	u := new(uuid.UUID)
	*u = uuid.Parse(keyJSON.ID)
	k.ID = *u
	pubkey, err := hex.DecodeString(keyJSON.PublicKey)
	if err != nil {
		return err
	}
	seckey, err := hex.DecodeString(keyJSON.SecretKey)
	if err != nil {
		return err
	}

	k.PublicKey.UnBufferPublicKey(pubkey)
	k.SecretKey.UnBufferSecretKey(seckey)

	return nil
}

func newKeyFromBLS(blsKey *bls.SecretKey) (*Key, error) {
	id := uuid.NewRandom()
	pubkey, err := blsKey.PublicKey()
	if err != nil {
		return nil, err
	}
	key := &Key{
		ID:        id,
		PublicKey: pubkey,
		SecretKey: blsKey,
	}
	return key, nil
}

// NewKey generates a new random key.
func NewKey(rand io.Reader) (*Key, error) {
	randBytes := make([]byte, 64)
	_, err := rand.Read(randBytes)
	if err != nil {
		return nil, fmt.Errorf("key generation: could not read from random source: %v", err)
	}
	secretKey := bls.GenerateKey(randBytes)

	return newKeyFromBLS(secretKey)
}

func storeNewRandomKey(ks keyStore, rand io.Reader, password string) error {
	key, err := NewKey(rand)
	if err != nil {
		return err
	}

	if err := ks.StoreKey(ks.JoinPath(keyFileName(key.PublicKey)), key, password); err != nil {
		zeroKey(key.SecretKey)
		return err
	}
	return nil
}

func writeKeyFile(file string, content []byte) error {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		newErr := f.Close()
		if newErr != nil {
			err = newErr
		}
		newErr = os.Remove(f.Name())
		if newErr != nil {
			err = newErr
		}
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), file)
}

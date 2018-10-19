package keystore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pborman/uuid"
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

type Key struct {
	Id uuid.UUID // Version 4 "random" for unique id not derived from key data
	// to simplify lookups we also store the address
	PublicKey *bls.PublicKey
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	SecretKey *bls.SecretKey
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
	Id        string `json:"id"`
}

type encryptedKeyJSON struct {
	PublicKey string     `json:"publickey"`
	Crypto    cryptoJSON `json:"crypto"`
	Id        string     `json:"id"`
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

func (k *Key) MarshalJSON() (j []byte, err error) {
	jStruct := plainKeyJSON{
		hex.EncodeToString(k.PublicKey.BufferedPublicKey()),
		hex.EncodeToString(k.SecretKey.BufferedSecretKey()),
		k.Id.String(),
	}
	j, err = json.Marshal(jStruct)
	return j, err
}

func (k *Key) UnmarshalJSON(j []byte) (err error) {
	keyJSON := new(plainKeyJSON)
	err = json.Unmarshal(j, &keyJSON)
	if err != nil {
		return err
	}

	u := new(uuid.UUID)
	*u = uuid.Parse(keyJSON.Id)
	k.Id = *u
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
		Id:        id,
		PublicKey: pubkey,
		SecretKey: blsKey,
	}
	return key, nil
}

func newKey(rand io.Reader) (*Key, error) {
	randBytes := make([]byte, 64)
	_, err := rand.Read(randBytes)
	if err != nil {
		panic("key generation: could not read from random source: " + err.Error())
	}
	secretKey := bls.GenerateKey(randBytes)

	return newKeyFromBLS(secretKey)
}

func storeNewKey(ks keyStore, rand io.Reader, auth string) (*Key, error) {
	key, err := newKey(rand)
	if err != nil {
		return nil, err
	}

	if err := ks.StoreKey(ks.JoinPath(keyFileName(key.PublicKey)), key, auth); err != nil {
		zeroKey(key.SecretKey)
		return nil, err
	}
	return key, err
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
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}

// keyFileName implements the naming convention for keyfiles:
// UTC--<created_at UTC ISO8601>-<address hex>
func keyFileName(pubkey *bls.PublicKey) string {
	ts := time.Now().UTC()
	return fmt.Sprintf("UTC--%s--%s", toISO8601(ts), hex.EncodeToString(pubkey.BufferedPublicKey()))
}

func toISO8601(t time.Time) string {
	var tz string
	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}

// zeroKey zeroes a private key in memory.
func zeroKey(k *bls.SecretKey) {
	b := k.K.Bits()
	for i := range b {
		b[i] = 0
	}
}

package keystore

import (
	"bytes"
	"crypto/aes"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/randentropy"
	"github.com/pborman/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

var (
	ErrDecrypt = errors.New("could not decrypt key with given passphrase")
)

type keyStorePassphrase struct {
	keysDirPath string
	scryptN     int
	scryptP     int
}

func (ks keyStorePassphrase) GetKey(filename, password string) (*Key, error) {
	// Load the key from the keystore and decrypt its contents
	// #nosec G304
	keyjson, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return DecryptKey(keyjson, password)
}

func (ks keyStorePassphrase) StoreKey(filename string, key *Key, auth string) error {
	keyjson, err := EncryptKey(key, auth, ks.scryptN, ks.scryptP)
	if err != nil {
		return err
	}
	return writeKeyFile(filename, keyjson)
}

func (ks keyStorePassphrase) JoinPath(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	return filepath.Join(ks.keysDirPath, filename)
}

// StoreKey generates a key, encrypts with 'auth' and stores in the given directory
func StoreKey(dir, password string, scryptN, scryptP int) error {
	_, err := storeNewKey(keyStorePassphrase{dir, scryptN, scryptP}, crand.Reader, password)
	return err
}

// EncryptKey encrypts a key using the specified scrypt parameters into a json
// blob that can be decrypted later on.
func EncryptKey(key *Key, password string, scryptN, scryptP int) ([]byte, error) {
	authArray := []byte(password)
	salt := randentropy.GetEntropyCSPRNG(32)
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}

	encryptKey := derivedKey[:16]
	keyBytes := math.PaddedBigBytes(key.SecretKey.K, 32)

	iv := randentropy.GetEntropyCSPRNG(aes.BlockSize) // 16
	cipherText, err := aesCTRXOR(encryptKey, keyBytes, iv)
	if err != nil {
		return nil, err
	}
	mac := crypto.Keccak256(derivedKey[16:32], cipherText)

	scryptParamsJSON := make(map[string]interface{}, 5)
	scryptParamsJSON["n"] = scryptN
	scryptParamsJSON["r"] = scryptR
	scryptParamsJSON["p"] = scryptP
	scryptParamsJSON["dklen"] = scryptDKLen
	scryptParamsJSON["salt"] = hex.EncodeToString(salt)

	cipherParamsJSON := cipherparamsJSON{
		IV: hex.EncodeToString(iv),
	}

	cryptoStruct := cryptoJSON{
		Cipher:       "aes-128-ctr",
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          keyHeaderKDF,
		KDFParams:    scryptParamsJSON,
		MAC:          hex.EncodeToString(mac),
	}
	encryptedJSON := encryptedKeyJSON{
		hex.EncodeToString(key.PublicKey.BufferedPublicKey()),
		cryptoStruct,
		key.Id.String(),
	}
	return json.Marshal(encryptedJSON)
}

// DecryptKey decrypts a key from a json blob, returning the private key itself.
func DecryptKey(keyjson []byte, password string) (*Key, error) {
	// Depending on the version try to parse one way or another
	var (
		keyBytes, keyId []byte
		err             error
	)

	k := new(encryptedKeyJSON)
	if err := json.Unmarshal(keyjson, k); err != nil {
		return nil, err
	}

	keyBytes, keyId, err = decryptKeyJSON(k, password)
	// Handle any decryption errors and return the key
	if err != nil {
		return nil, err
	}

	key := &bls.SecretKey{}
	key.UnBufferSecretKey(keyBytes)
	pubkey, err := key.PublicKey()
	if err != nil {
		return nil, err
	}

	return &Key{
		Id:        uuid.UUID(keyId),
		PublicKey: pubkey,
		SecretKey: key,
	}, nil
}

func decryptKeyJSON(keyProtected *encryptedKeyJSON, auth string) (keyBytes []byte, keyId []byte, err error) {
	keyId = uuid.Parse(keyProtected.Id)
	mac, err := hex.DecodeString(keyProtected.Crypto.MAC)
	if err != nil {
		return nil, nil, err
	}

	iv, err := hex.DecodeString(keyProtected.Crypto.CipherParams.IV)
	if err != nil {
		return nil, nil, err
	}

	cipherText, err := hex.DecodeString(keyProtected.Crypto.CipherText)
	if err != nil {
		return nil, nil, err
	}

	derivedKey, err := getKDFKey(keyProtected.Crypto, auth)
	if err != nil {
		return nil, nil, err
	}

	calculatedMAC := crypto.Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, nil, ErrDecrypt
	}

	plainText, err := aesCBCDecrypt(crypto.Keccak256(derivedKey[:16])[:16], cipherText, iv)
	if err != nil {
		return nil, nil, err
	}
	return plainText, keyId, err
}

func getKDFKey(cryptoJSON cryptoJSON, auth string) ([]byte, error) {
	authArray := []byte(auth)
	salt, err := hex.DecodeString(cryptoJSON.KDFParams["salt"].(string))
	if err != nil {
		return nil, err
	}
	dkLen := ensureInt(cryptoJSON.KDFParams["dklen"])

	if cryptoJSON.KDF == keyHeaderKDF {
		n := ensureInt(cryptoJSON.KDFParams["n"])
		r := ensureInt(cryptoJSON.KDFParams["r"])
		p := ensureInt(cryptoJSON.KDFParams["p"])
		return scrypt.Key(authArray, salt, n, r, p, dkLen)

	} else if cryptoJSON.KDF == "pbkdf2" {
		c := ensureInt(cryptoJSON.KDFParams["c"])
		prf := cryptoJSON.KDFParams["prf"].(string)
		if prf != "hmac-sha256" {
			return nil, fmt.Errorf("Unsupported PBKDF2 PRF: %s", prf)
		}
		key := pbkdf2.Key(authArray, salt, c, dkLen, sha256.New)
		return key, nil
	}

	return nil, fmt.Errorf("Unsupported KDF: %s", cryptoJSON.KDF)
}

func RetrieveJson(directory string, password string) ([]byte, error) {
	f, err := os.Open(directory)
	if err != nil {
		return nil, err
	}

	keyjson, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return keyjson, nil
}

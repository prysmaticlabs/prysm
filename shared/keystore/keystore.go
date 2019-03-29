// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// Modified by Prysmatic Labs 2018
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package keystore

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/pborman/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

var (
	// ErrDecrypt is the standard error message when decryption is a failure.
	ErrDecrypt = errors.New("could not decrypt key with given passphrase")
)

// Store defines a keystore with a directory path and scrypt values.
type Store struct {
	keysDirPath string
	scryptN     int
	scryptP     int
}

// RetrievePubKey retrieves the public key from the keystore.
func RetrievePubKey(directory string, password string) (*bls.PublicKey, error) {
	ks := Store{
		keysDirPath: directory,
		scryptN:     StandardScryptN,
		scryptP:     StandardScryptP,
	}
	key, err := ks.GetKey(ks.keysDirPath, password)
	return key.PublicKey, err
}

// NewKeystore from a directory.
func NewKeystore(directory string) Store {
	return Store{
		keysDirPath: directory,
		scryptN:     StandardScryptN,
		scryptP:     StandardScryptP,
	}
}

// GetKey from file using the filename path and a decryption password.
func (ks Store) GetKey(filename, password string) (*Key, error) {
	// Load the key from the keystore and decrypt its contents
	// #nosec G304
	keyjson, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return DecryptKey(keyjson, password)
}

// GetKeys from directory using the prefix to filter relevant files
// and a decryption password.
func (ks Store) GetKeys(directory, fileprefix, password string) (map[string]*Key, error) {
	// Load the key from the keystore and decrypt its contents
	// #nosec G304
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]*Key)
	for _, f := range files {
		n := f.Name()
		filePath := filepath.Join(directory, n)
		filePath = filepath.Clean(filePath)
		cp := strings.Contains(n, strings.TrimPrefix(fileprefix, "/"))
		if f.Mode().IsRegular() && cp {
			// #nosec G304
			keyjson, err := ioutil.ReadFile(filePath)
			if err != nil {
				return nil, err
			}
			key, err := DecryptKey(keyjson, password)
			if err != nil {
				return nil, err
			}
			keys[hex.EncodeToString(key.PublicKey.Marshal())] = key
		}
	}
	return keys, nil
}

// StoreKey in filepath and encrypt it with a password.
func (ks Store) StoreKey(filename string, key *Key, auth string) error {
	keyjson, err := EncryptKey(key, auth, ks.scryptN, ks.scryptP)
	if err != nil {
		return err
	}
	return writeKeyFile(filename, keyjson)
}

// JoinPath joins the filename with the keystore directory path.
func (ks Store) JoinPath(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	return filepath.Join(ks.keysDirPath, filename)
}

// StoreRandomKey generates a key, encrypts with 'auth' and stores in the given directory
func StoreRandomKey(dir, password string, scryptN, scryptP int) error {
	err := storeNewRandomKey(Store{dir, scryptN, scryptP}, rand.Reader, password)
	return err
}

// EncryptKey encrypts a key using the specified scrypt parameters into a json
// blob that can be decrypted later on.
func EncryptKey(key *Key, password string, scryptN, scryptP int) ([]byte, error) {
	authArray := []byte(password)
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}

	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}

	encryptKey := derivedKey[:16]
	keyBytes := key.SecretKey.Marshal()

	iv := make([]byte, aes.BlockSize) // 16
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, errors.New("reading from crypto/rand failed: " + err.Error())
	}

	cipherText, err := aesCTRXOR(encryptKey, keyBytes, iv)
	if err != nil {
		return nil, err
	}

	mac := Keccak256(derivedKey[16:32], cipherText)

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
		hex.EncodeToString(key.PublicKey.Marshal()),
		cryptoStruct,
		key.ID.String(),
	}
	return json.Marshal(encryptedJSON)
}

// DecryptKey decrypts a key from a json blob, returning the private key itself.
func DecryptKey(keyjson []byte, password string) (*Key, error) {
	var keyBytes, keyID []byte
	var err error

	k := new(encryptedKeyJSON)
	if err := json.Unmarshal(keyjson, k); err != nil {
		return nil, err
	}

	keyBytes, keyID, err = decryptKeyJSON(k, password)
	// Handle any decryption errors and return the key
	if err != nil {
		return nil, err
	}

	secretKey, err := bls.SecretKeyFromBytes(keyBytes)
	if err != nil {
		return nil, err
	}

	return &Key{
		ID:        uuid.UUID(keyID),
		PublicKey: secretKey.PublicKey(),
		SecretKey: secretKey,
	}, nil
}

func decryptKeyJSON(keyProtected *encryptedKeyJSON, auth string) (keyBytes []byte, keyID []byte, err error) {
	keyID = uuid.Parse(keyProtected.ID)
	if keyProtected.Crypto.Cipher != "aes-128-ctr" {
		return nil, nil, fmt.Errorf("cipher not supported: %v", keyProtected.Crypto.Cipher)
	}

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

	calculatedMAC := Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, nil, ErrDecrypt
	}

	plainText, err := aesCTRXOR(derivedKey[:16], cipherText, iv)
	if err != nil {
		return nil, nil, err
	}
	return plainText, keyID, nil
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
			return nil, fmt.Errorf("unsupported PBKDF2 PRF: %s", prf)
		}
		key := pbkdf2.Key(authArray, salt, c, dkLen, sha256.New)
		return key, nil
	}

	return nil, fmt.Errorf("unsupported KDF: %s", cryptoJSON.KDF)
}

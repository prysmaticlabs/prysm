package keystore

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"io/ioutil"
	"os"
)

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

func DecryptKeystore(keyjson []byte, password string) (ecdsa.PublicKey, error) {
	key, err := keystore.DecryptKey(keyjson, password)
	if err != nil {
		return ecdsa.PublicKey{}, err
	}
	return key.PrivateKey.PublicKey, nil
}

func EncryptKeystore(key *keystore.Key, password string) ([]byte, error) {
	encryptedkey, err := keystore.EncryptKey(key, password, 8, 1)
	if err != nil {
		return nil, err
	}
	return encryptedkey, nil
}

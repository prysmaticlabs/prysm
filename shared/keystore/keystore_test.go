package keystore

import (
	"bytes"
	"github.com/pborman/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"math/big"
	"testing"
)

func TestEncryptDecryptKey(t *testing.T) {
	newID := uuid.NewRandom()
	keyValue := big.NewInt(10)
	password := "test"

	key := &Key{
		ID: newID,
		SecretKey: &bls.SecretKey{
			K: keyValue,
		},
	}

	encryptedStore, err := EncryptKey(key, password, StandardScryptN, StandardScryptP)
	if err != nil {
		t.Fatalf("unable to encrypt key %v", err)
	}

	newkey, err := DecryptKey(encryptedStore, password)
	if err != nil {
		t.Fatalf("unable to decrypt keystore %v", err)
	}

	if !bytes.Equal(newkey.ID, newID) {
		t.Fatalf("decrypted key's uuid doesn't match %v", newkey.ID)
	}

	if newkey.SecretKey.K.Cmp(keyValue) != 0 {
		t.Fatalf("decrypted key's value is not equal %v", newkey.SecretKey.K)
	}

}

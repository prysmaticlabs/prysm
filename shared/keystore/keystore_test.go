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
	keyValue := big.NewInt(1e16)
	password := "test"

	key := &Key{
		ID: newID,
		SecretKey: &bls.SecretKey{
			K: keyValue,
		},
	}

	keyjson, err := EncryptKey(key, password, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("unable to encrypt key %v", err)
	}

	newkey, err := DecryptKey(keyjson, password)
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

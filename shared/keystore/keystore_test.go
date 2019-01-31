package keystore

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"os"
	"testing"

	"github.com/pborman/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

func TestStoreandGetKey(t *testing.T) {
	tmpdir := os.TempDir()
	filedir := tmpdir + "/keystore"
	ks := &Store{
		keysDirPath: filedir,
		scryptN:     LightScryptN,
		scryptP:     LightScryptP,
	}

	key, err := NewKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed %v", err)
	}

	if err := ks.StoreKey(filedir, key, "password"); err != nil {
		t.Fatalf("unable to store key %v", err)
	}

	newkey, err := ks.GetKey(filedir, "password")
	if err != nil {
		t.Fatalf("unable to get key %v", err)
	}

	if newkey.SecretKey.K.Cmp(key.SecretKey.K) != 0 {
		t.Fatalf("retrieved secret keys are not equal %v , %v", newkey.SecretKey.K, key.SecretKey.K)
	}

	if err := os.RemoveAll(filedir); err != nil {
		t.Errorf("unable to remove temporary files %v", err)
	}
}
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

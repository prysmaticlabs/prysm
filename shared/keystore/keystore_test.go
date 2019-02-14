package keystore

import (
	"bytes"
	"crypto/rand"
	"os"
	"testing"

	"github.com/pborman/uuid"
	bls "github.com/prysmaticlabs/go-bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestStoreandGetKey(t *testing.T) {
	tmpdir := testutil.TempDir()
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

	if !newkey.SecretKey.IsEqual(key.SecretKey) {
		t.Fatalf("retrieved secret keys are not equal %v , %v", newkey.SecretKey.LittleEndian(), key.SecretKey.LittleEndian())
	}

	if err := os.RemoveAll(filedir); err != nil {
		t.Errorf("unable to remove temporary files %v", err)
	}
}
func TestEncryptDecryptKey(t *testing.T) {
	newID := uuid.NewRandom()
	blsKey := &bls.SecretKey{}
	blsKey.SetByCSPRNG()
	password := "test"

	key := &Key{
		ID:        newID,
		SecretKey: blsKey,
		PublicKey: blsKey.GetPublicKey(),
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

	if !newkey.SecretKey.IsEqual(blsKey) {
		t.Fatalf("decrypted key's value is not equal %v", newkey.SecretKey.LittleEndian())
	}

}

package keystore

import (
	"bytes"
	"crypto/rand"
	"os"
	"testing"

	"github.com/pborman/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestStoreAndGetKey(t *testing.T) {
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

	if !bytes.Equal(newkey.SecretKey.Marshal(), key.SecretKey.Marshal()) {
		t.Fatalf("retrieved secret keys are not equal %v , %v", newkey.SecretKey.Marshal(), key.SecretKey.Marshal())
	}

	if err := os.RemoveAll(filedir); err != nil {
		t.Errorf("unable to remove temporary files %v", err)
	}
}
func TestEncryptDecryptKey(t *testing.T) {
	newID := uuid.NewRandom()
	b := []byte("hi")
	b32 := bytesutil.ToBytes32(b)
	password := "test"

	pk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	key := &Key{
		ID:        newID,
		SecretKey: pk,
		PublicKey: pk.PublicKey(),
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

	expected := pk.Marshal()
	if !bytes.Equal(newkey.SecretKey.Marshal(), expected) {
		t.Fatalf("decrypted key's value is not equal %v", newkey.SecretKey.Marshal())
	}

}

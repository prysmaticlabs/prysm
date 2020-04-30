package keystore

import (
	"bytes"
	"os"
	"testing"

	"github.com/pborman/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestStoreAndGetKey(t *testing.T) {
	tempDir := testutil.TempDir() + "/keystore"
	defer teardownTempKeystore(t, tempDir)
	ks := &Store{
		keysDirPath: tempDir,
		scryptN:     LightScryptN,
		scryptP:     LightScryptP,
	}

	key, err := NewKey()
	if err != nil {
		t.Fatalf("key generation failed %v", err)
	}

	if err := ks.StoreKey(tempDir, key, "password"); err != nil {
		t.Fatalf("unable to store key %v", err)
	}

	newkey, err := ks.GetKey(tempDir, "password")
	if err != nil {
		t.Fatalf("unable to get key %v", err)
	}

	if !bytes.Equal(newkey.SecretKey.Marshal(), key.SecretKey.Marshal()) {
		t.Fatalf("retrieved secret keys are not equal %v , %v", newkey.SecretKey.Marshal(), key.SecretKey.Marshal())
	}
}

func TestStoreAndGetKeys(t *testing.T) {
	tempDir := testutil.TempDir() + "/keystore"
	defer teardownTempKeystore(t, tempDir)
	ks := &Store{
		keysDirPath: tempDir,
		scryptN:     LightScryptN,
		scryptP:     LightScryptP,
	}

	key, err := NewKey()
	if err != nil {
		t.Fatalf("key generation failed %v", err)
	}

	if err := ks.StoreKey(tempDir+"/test-1", key, "password"); err != nil {
		t.Fatalf("unable to store key %v", err)
	}
	key2, err := NewKey()
	if err != nil {
		t.Fatalf("key generation failed %v", err)
	}
	if err := ks.StoreKey(tempDir+"/test-2", key2, "password"); err != nil {
		t.Fatalf("unable to store key %v", err)
	}
	newkeys, err := ks.GetKeys(tempDir, "test", "password", false)
	if err != nil {
		t.Fatalf("unable to get key %v", err)
	}
	for _, s := range newkeys {
		if !bytes.Equal(s.SecretKey.Marshal(), key.SecretKey.Marshal()) && !bytes.Equal(s.SecretKey.Marshal(), key2.SecretKey.Marshal()) {
			t.Fatalf("retrieved secret keys are not equal %v ", s.SecretKey.Marshal())
		}

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

func TestGetSymlinkedKeys(t *testing.T) {
	tempDir := testutil.TempDir() + "/keystore"
	defer teardownTempKeystore(t, tempDir)
	ks := &Store{
		scryptN: LightScryptN,
		scryptP: LightScryptP,
	}

	key, err := NewKey()
	if err != nil {
		t.Fatalf("key generation failed %v", err)
	}

	if err := ks.StoreKey(tempDir+"/files/test-1", key, "password"); err != nil {
		t.Fatalf("unable to store key %v", err)
	}

	if err := os.Symlink(tempDir+"/files/test-1", tempDir+"/test-1"); err != nil {
		t.Fatalf("unable to create symlink: %v", err)
	}

	newkeys, err := ks.GetKeys(tempDir, "test", "password", false)
	if err != nil {
		t.Fatalf("unable to get key %v", err)
	}
	if len(newkeys) != 1 {
		t.Errorf("unexpected number of keys returned, want: %d, got: %d", 1, len(newkeys))
	}
	for _, s := range newkeys {
		if !bytes.Equal(s.SecretKey.Marshal(), key.SecretKey.Marshal()) {
			t.Fatalf("retrieved secret keys are not equal %v ", s.SecretKey.Marshal())
		}
	}
}

// teardownTempKeystore removes temporary directory used for keystore testing.
func teardownTempKeystore(t *testing.T, tempDir string) {
	if err := os.RemoveAll(tempDir); err != nil {
		t.Logf("unable to remove temporary files: %v", err)
	}
}

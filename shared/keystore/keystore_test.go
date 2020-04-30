package keystore

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/pborman/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestStoreAndGetKey(t *testing.T) {
	tempDir, teardown := setupTempKeystoreDir(t)
	defer teardown()
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

	decryptedKey, err := ks.GetKey(tempDir, "password")
	if err != nil {
		t.Fatalf("unable to get key %v", err)
	}

	if !bytes.Equal(decryptedKey.SecretKey.Marshal(), key.SecretKey.Marshal()) {
		t.Fatalf("retrieved secret keys are not equal %v , %v", decryptedKey.SecretKey.Marshal(), key.SecretKey.Marshal())
	}
}

func TestStoreAndGetKeys(t *testing.T) {
	tempDir, teardown := setupTempKeystoreDir(t)
	defer teardown()
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
	decryptedKeys, err := ks.GetKeys(tempDir, "test", "password", false)
	if err != nil {
		t.Fatalf("unable to get key %v", err)
	}
	for _, s := range decryptedKeys {
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

	keyJSON, err := EncryptKey(key, password, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatalf("unable to encrypt key %v", err)
	}

	decryptedKey, err := DecryptKey(keyJSON, password)
	if err != nil {
		t.Fatalf("unable to decrypt keystore %v", err)
	}

	if !bytes.Equal(decryptedKey.ID, newID) {
		t.Fatalf("decrypted key's uuid doesn't match %v", decryptedKey.ID)
	}

	expected := pk.Marshal()
	if !bytes.Equal(decryptedKey.SecretKey.Marshal(), expected) {
		t.Fatalf("decrypted key's value is not equal %v", decryptedKey.SecretKey.Marshal())
	}
}

func TestGetSymlinkedKeys(t *testing.T) {
	tempDir, teardown := setupTempKeystoreDir(t)
	defer teardown()
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

	decryptedKeys, err := ks.GetKeys(tempDir, "test", "password", false)
	if err != nil {
		t.Fatalf("unable to get key %v", err)
	}
	if len(decryptedKeys) != 1 {
		t.Errorf("unexpected number of keys returned, want: %d, got: %d", 1, len(decryptedKeys))
	}
	for _, s := range decryptedKeys {
		if !bytes.Equal(s.SecretKey.Marshal(), key.SecretKey.Marshal()) {
			t.Fatalf("retrieved secret keys are not equal %v ", s.SecretKey.Marshal())
		}
	}
}

// setupTempKeystoreDir creates temporary directory for storing keystore files.
func setupTempKeystoreDir(t *testing.T) (string, func()) {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("could not generate random file path: %v", err)
	}
	tempDir := path.Join(testutil.TempDir(), fmt.Sprintf("%d", randPath), "keystore")

	return tempDir, func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("unable to remove temporary files: %v", err)
		}
	}
}

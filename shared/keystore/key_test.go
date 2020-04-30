package keystore

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/pborman/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func TestMarshalAndUnmarshal(t *testing.T) {
	testID := uuid.NewRandom()
	blsKey := bls.RandKey()

	key := &Key{
		ID:        testID,
		SecretKey: blsKey,
		PublicKey: blsKey.PublicKey(),
	}
	marshalledObject, err := key.MarshalJSON()
	if err != nil {
		t.Fatalf("unable to marshall key %v", err)
	}
	newKey := &Key{
		ID:        []byte{},
		SecretKey: blsKey,
		PublicKey: blsKey.PublicKey(),
	}

	err = newKey.UnmarshalJSON(marshalledObject)
	if err != nil {
		t.Fatalf("unable to unmarshal object %v", err)
	}

	if !bytes.Equal(newKey.ID, testID) {
		t.Fatalf("retrieved id not the same as pre serialized id: %v ", newKey.ID)
	}
}

func TestStoreRandomKey(t *testing.T) {
	tempDir, teardown := setupTempKeystoreDir(t)
	defer teardown()
	ks := &Store{
		keysDirPath: tempDir,
		scryptN:     LightScryptN,
		scryptP:     LightScryptP,
	}

	if err := storeNewRandomKey(ks, "password"); err != nil {
		t.Fatalf("storage of random key unsuccessful %v", err)
	}
}

func TestNewKeyFromBLS(t *testing.T) {
	b := []byte("hi")
	b32 := bytesutil.ToBytes32(b)
	blskey, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	key, err := NewKeyFromBLS(blskey)
	if err != nil {
		t.Fatalf("could not get new key from bls %v", err)
	}

	expected := blskey.Marshal()
	if !bytes.Equal(expected, key.SecretKey.Marshal()) {
		t.Fatalf("secret key is not of the expected value %d", key.SecretKey.Marshal())
	}

	_, err = NewKey()
	if err != nil {
		t.Fatalf("random key unable to be generated: %v", err)
	}
}

func TestWriteFile(t *testing.T) {
	tempDir, teardown := setupTempKeystoreDir(t)
	defer teardown()

	testKeystore := []byte{'t', 'e', 's', 't'}

	err := writeKeyFile(tempDir, testKeystore)
	if err != nil {
		t.Fatalf("unable to write file %v", err)
	}

	keystore, err := ioutil.ReadFile(tempDir)
	if err != nil {
		t.Fatalf("unable to retrieve file %v", err)
	}

	if !bytes.Equal(keystore, testKeystore) {
		t.Fatalf("retrieved keystore is not the same %v", keystore)
	}
}

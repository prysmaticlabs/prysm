package keystore

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"

	"github.com/pborman/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestMarshalAndUnmarshal(t *testing.T) {
	testID := uuid.NewRandom()
	blsKey, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
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
		t.Fatalf("unable to unmarshall object %v", err)
	}

	if !bytes.Equal([]byte(newKey.ID), []byte(testID)) {
		t.Fatalf("retrieved id not the same as pre serialized id: %v ", newKey.ID)
	}
}

func TestStoreRandomKey(t *testing.T) {
	tmpdir := testutil.TempDir()
	filedir := tmpdir + "/keystore"
	ks := &Store{
		keysDirPath: filedir,
		scryptN:     LightScryptN,
		scryptP:     LightScryptP,
	}

	reader := rand.Reader

	if err := storeNewRandomKey(ks, reader, "password"); err != nil {
		t.Fatalf("storage of random key unsuccessful %v", err)
	}

	if err := os.RemoveAll(filedir); err != nil {
		t.Errorf("unable to remove temporary files %v", err)
	}
}

func TestNewKeyFromBLS(t *testing.T) {
	b := []byte("hi")
	b32 := bytesutil.ToBytes32(b)
	blskey, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	key, err := newKeyFromBLS(blskey)
	if err != nil {
		t.Fatalf("could not get new key from bls %v", err)
	}

	expected := blskey.Marshal()
	if !bytes.Equal(expected, key.SecretKey.Marshal()) {
		t.Fatalf("secret key is not of the expected value %d", key.SecretKey.Marshal())
	}

	reader := rand.Reader

	_, err = NewKey(reader)
	if err != nil {
		t.Fatalf("random key unable to be generated: %v", err)
	}
}

func TestWriteFile(t *testing.T) {
	tmpdir := testutil.TempDir()
	filedir := tmpdir + "/keystore"

	testKeystore := []byte{'t', 'e', 's', 't'}

	err := writeKeyFile(filedir, testKeystore)
	if err != nil {
		t.Fatalf("unable to write file %v", err)
	}

	keystore, err := ioutil.ReadFile(filedir)
	if err != nil {
		t.Fatalf("unable to retrieve file %v", err)
	}

	if !bytes.Equal(keystore, testKeystore) {
		t.Fatalf("retrieved keystore is not the same %v", keystore)
	}

	if err := os.RemoveAll(filedir); err != nil {
		t.Errorf("unable to remove temporary files %v", err)
	}
}

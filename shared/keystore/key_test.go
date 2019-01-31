package keystore

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/pborman/uuid"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

func TestMarshalAndUnmarshal(t *testing.T) {
	testID := uuid.NewRandom()
	blsKey := &bls.SecretKey{
		K: big.NewInt(10),
	}
	key := &Key{
		ID:        testID,
		SecretKey: blsKey,
	}
	marshalledObject, err := key.MarshalJSON()
	if err != nil {
		t.Fatalf("unable to marshall key %v", err)
	}
	newKey := &Key{
		ID: []byte{},
		SecretKey: &bls.SecretKey{
			K: big.NewInt(0),
		},
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
	tmpdir := os.TempDir()
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
	blskey := &bls.SecretKey{
		K: big.NewInt(20),
	}

	key, err := newKeyFromBLS(blskey)
	if err != nil {
		t.Fatalf("could not get new key from bls %v", err)
	}

	expectedNum := big.NewInt(20)

	if expectedNum.Cmp(key.SecretKey.K) != 0 {
		t.Fatalf("secret key is not of the expected value %d", key.SecretKey.K)
	}

	reader := rand.Reader

	_, err = NewKey(reader)
	if err != nil {
		t.Fatalf("random key unable to be generated: %v", err)
	}

}

func TestWriteFile(t *testing.T) {
	tmpdir := os.TempDir()
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

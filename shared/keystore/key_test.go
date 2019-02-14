package keystore

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"io/ioutil"
	"os"
	"testing"

	"github.com/pborman/uuid"
	bls "github.com/prysmaticlabs/go-bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestMarshalAndUnmarshal(t *testing.T) {
	testID := uuid.NewRandom()
	blsKey := &bls.SecretKey{}
	blsKey.SetValue(10)
	key := &Key{
		ID:        testID,
		PublicKey: blsKey.GetPublicKey(),
		SecretKey: blsKey,
	}
	marshalledObject, err := key.MarshalJSON()
	if err != nil {
		t.Fatalf("unable to marshall key %v", err)
	}

	newKey := &Key{
		ID:        []byte{},
		SecretKey: &bls.SecretKey{},
		PublicKey: &bls.PublicKey{},
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
	blskey := &bls.SecretKey{}

	expectedNum := int64(20)
	blskey.SetValue(expectedNum)

	key := newKeyFromBLS(blskey)

	keyBuffer := make([]byte, len(key.SecretKey.LittleEndian()))
	binary.LittleEndian.PutUint64(keyBuffer, uint64(expectedNum))

	if !bytes.Equal(key.SecretKey.LittleEndian(), keyBuffer) {
		t.Fatalf("secret key is not of the expected value %v , %v", key.SecretKey.LittleEndian(), keyBuffer)
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

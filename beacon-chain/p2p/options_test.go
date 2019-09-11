package p2p

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestPrivateKeyLoading(t *testing.T) {
	file, err := ioutil.TempFile(testutil.TempDir(), "key")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())
	key, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		t.Fatalf("Could not generate key: %v", err)
	}
	marshalledKey, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		t.Fatalf("Could not marshal key %v", err)
	}
	encodedKey := crypto.ConfigEncodeKey(marshalledKey)

	err = ioutil.WriteFile(file.Name(), []byte(encodedKey), 0600)
	if err != nil {
		t.Fatalf("Could not write key to file: %v", err)
	}
	log.WithField("file", file.Name()).WithField("key", encodedKey).Info("Wrote key to file")
	cfg := &Config{
		PrivateKey: file.Name(),
		Encoding:   "ssz",
	}
	pKey, err := privKey(cfg)
	if err != nil {
		t.Fatalf("Could not apply option: %v", err)
	}
	newPkey := convertToInterfacePrivkey(pKey)
	rawBytes, err := key.Raw()
	if err != nil {
		t.Fatal(err)
	}
	newRaw, _ := newPkey.Raw()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(newRaw, rawBytes) {
		t.Errorf("Private keys do not match got %#x but wanted %#x", rawBytes, newRaw)
	}
}

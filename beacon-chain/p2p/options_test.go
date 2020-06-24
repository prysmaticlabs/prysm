package p2p

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestPrivateKeyLoading(t *testing.T) {
	file, err := ioutil.TempFile(testutil.TempDir(), "key")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := os.Remove(file.Name()); err != nil {
			t.Log(err)
		}
	}()
	key, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		t.Fatalf("Could not generate key: %v", err)
	}
	raw, err := key.Raw()
	if err != nil {
		panic(err)
	}
	out := hex.EncodeToString(raw)

	err = ioutil.WriteFile(file.Name(), []byte(out), params.BeaconIoConfig().FilePermission)
	if err != nil {
		t.Fatalf("Could not write key to file: %v", err)
	}
	log.WithField("file", file.Name()).WithField("key", out).Info("Wrote key to file")
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
	newRaw, err := newPkey.Raw()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(newRaw, rawBytes) {
		t.Errorf("Private keys do not match got %#x but wanted %#x", rawBytes, newRaw)
	}
}

package p2p

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"

	curve "github.com/ethereum/go-ethereum/crypto"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestPrivateKeyLoading(t *testing.T) {
	file, err := ioutil.TempFile(testutil.TempDir(), "key")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())
	key, err := ecdsa.GenerateKey(curve.S256(), rand.Reader)
	if err != nil {
		t.Fatalf("Could not generate key: %v", err)
	}
	keyStr := hex.EncodeToString(curve.FromECDSA(key))
	err = ioutil.WriteFile(file.Name(), []byte(keyStr), 0600)
	if err != nil {
		t.Fatalf("Could not write key to file: %v", err)
	}
	log.WithField("file", file.Name()).WithField("key", keyStr).Info("Wrote key to file")
	cfg := &Config{
		PrivateKey: file.Name(),
		Encoding: "ssz",
	}
	pKey, err := privKey(cfg)
	if err != nil {
		t.Fatalf("Could not apply option: %v", err)
	}
	newEncoded := hex.EncodeToString(curve.FromECDSA(pKey))
	if newEncoded != keyStr {
		t.Error("Private keys do not match")
	}
}

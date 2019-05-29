package p2p

import (
	"io/ioutil"
	"os"
	"testing"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p/config"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestBuildOptions(t *testing.T) {
	opts := buildOptions(&ServerConfig{})

	_ = opts
}

func TestPrivateKeyLoading(t *testing.T) {
	file, err := ioutil.TempFile(testutil.TempDir(), "key")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())
	key, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		t.Fatalf("Could not generate key: %v", err)
	}
	marshaled, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}
	keyStr := crypto.ConfigEncodeKey(marshaled)

	err = ioutil.WriteFile(file.Name(), []byte(keyStr), 0600)
	if err != nil {
		t.Fatalf("Could not write key to file: %v", err)
	}
	log.WithField("file", file.Name()).WithField("key", keyStr).Info("Wrote key to file")

	var cfg config.Config
	err = cfg.Apply(privKey(file.Name()))
	if err != nil {
		t.Fatalf("Could not apply option: %v", err)
	}
	newMarshaled, _ := crypto.MarshalPrivateKey(cfg.PeerKey)
	newEncoded := crypto.ConfigEncodeKey(newMarshaled)
	if newEncoded != keyStr {
		t.Error("Private keys do not match")
	}
}

package keymanager_test

import (
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
)

func TestDirectListValidatingKeysNil(t *testing.T) {
	direct := keymanager.NewDirect(nil)
	keys, err := direct.FetchValidatingKeys()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("Incorrect number of keys returned; expected 0, received %d", len(keys))
	}
}

func TestDirectListValidatingKeysSingle(t *testing.T) {
	sks := make([]bls.SecretKey, 0)
	sks = append(sks, bls.RandKey())
	direct := keymanager.NewDirect(sks)
	keys, err := direct.FetchValidatingKeys()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("Incorrect number of keys returned; expected 1, received %d", len(keys))
	}
}

func TestDirectListValidatingKeysMultiple(t *testing.T) {
	sks := make([]bls.SecretKey, 0)
	numKeys := 256
	for i := 0; i < numKeys; i++ {
		sks = append(sks, bls.RandKey())
	}
	direct := keymanager.NewDirect(sks)
	keys, err := direct.FetchValidatingKeys()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(keys) != numKeys {
		t.Errorf("Incorrect number of keys returned; expected %d, received %d", numKeys, len(keys))
	}
}

func TestSignNoSuchKey(t *testing.T) {
	sks := make([]bls.SecretKey, 0)
	//	sks = append(sks, bls.RandKey())
	direct := keymanager.NewDirect(sks)

	sig, err := direct.Sign([48]byte{}, [32]byte{})
	if err != keymanager.ErrNoSuchKey {
		t.Fatalf("Incorrect error: expected %v, received %v", keymanager.ErrNoSuchKey, err)
	}
	fmt.Printf("%v\n", sig)
}

func TestSign(t *testing.T) {
	sks := make([]bls.SecretKey, 0)
	sks = append(sks, bls.RandKey())
	direct := keymanager.NewDirect(sks)

	pubKey := bytesutil.ToBytes48(sks[0].PublicKey().Marshal())
	msg := [32]byte{}
	sig, err := direct.Sign(pubKey, msg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !sig.Verify(sks[0].PublicKey(), bytesutil.FromBytes32(msg)) {
		t.Fatal("Failed to verify generated signature")
	}
}

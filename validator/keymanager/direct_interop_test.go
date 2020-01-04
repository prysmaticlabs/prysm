package keymanager_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
)

func TestInteropListValidatingKeysZero(t *testing.T) {
	_, err := keymanager.NewInterop(0, 0)
	if err == nil {
		t.Fatal("Missing expected error")
	}
	if err.Error() != "failed to generate keys: input length must be greater than 0" {
		t.Errorf("Incorrect value for error; expected \"failed to generate keys: input length must be greater than 0\", received %d", err)
	}
}

func TestInteropListValidatingKeysSingle(t *testing.T) {
	direct, err := keymanager.NewInterop(1, 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	keys, err := direct.FetchValidatingKeys()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("Incorrect number of keys returned; expected 1, received %d", len(keys))
	}

	pkBytes, err := hex.DecodeString("25295f0d1d592a90b333e26e85149708208e9f8e8bc18f6c77bd62f8ad7a6866")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	privateKey, err := bls.SecretKeyFromBytes(pkBytes)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !bytes.Equal(privateKey.PublicKey().Marshal(), keys[0][:]) {
		t.Fatalf("Public k 0 incorrect; expected %x, received %x", privateKey.PublicKey().Marshal(), keys[0])
	}
}

func TestInteropListValidatingKeysOffset(t *testing.T) {
	direct, err := keymanager.NewInterop(1, 9)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	keys, err := direct.FetchValidatingKeys()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("Incorrect number of keys returned; expected 1, received %d", len(keys))
	}

	pkBytes, err := hex.DecodeString("2b3b88a041168a1c4cd04bdd8de7964fd35238f95442dc678514f9dadb81ec34")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	privateKey, err := bls.SecretKeyFromBytes(pkBytes)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !bytes.Equal(privateKey.PublicKey().Marshal(), keys[0][:]) {
		t.Fatalf("Public k 0 incorrect; expected %x, received %x", privateKey.PublicKey().Marshal(), keys[0])
	}
}

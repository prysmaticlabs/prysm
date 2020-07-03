package v1_test

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v1"
)

func TestInteropListValidatingKeysZero(t *testing.T) {
	_, _, err := keymanager.NewInterop("")
	if err == nil {
		t.Fatal("Missing expected error")
	}
	expectedErr := "unexpected end of JSON input"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Incorrect value for error; expected %q, received %v", expectedErr, err)
	}
}

func TestInteropListValidatingKeysEmptyJSON(t *testing.T) {
	_, _, err := keymanager.NewInterop("{}")
	if err == nil {
		t.Fatal("Missing expected error")
	}
	expectedErr := "input length must be greater than 0"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Incorrect value for error; expected %q, received %v", expectedErr, err)
	}
}

func TestInteropListValidatingKeysSingle(t *testing.T) {
	direct, _, err := keymanager.NewInterop(`{"keys":1}`)
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
	direct, _, err := keymanager.NewInterop(`{"keys":1,"offset":9}`)
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

package direct

import (
	"bytes"
	"testing"

	"github.com/tyler-smith/go-bip39"
)

func TestMnemonic_Generate_CanRecover(t *testing.T) {
	generator := &EnglishMnemonicGenerator{}
	data := make([]byte, 32)
	copy(data, []byte("hello-world"))
	phrase, err := generator.Generate(data)
	if err != nil {
		t.Fatal(err)
	}
	entropy, err := bip39.EntropyFromMnemonic(phrase)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(entropy, data) {
		t.Errorf("Expected to recover original data: %v, received %v", data, entropy)
	}
}

package testutil

import (
	"crypto/rand"
	"testing"
)

func Random32Bytes(t *testing.T) []byte {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

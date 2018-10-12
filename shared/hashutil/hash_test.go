package hashutil

import (
	"testing"
)

func TestHash(t *testing.T) {
	hash := Hash([]byte{0})
	test := Hash([]byte{0})
	if hash != test {
		t.Fatalf("expected hash and computed hash are not equal %d, %d", test, hash)
	}
	test = Hash([]byte{1})
	if hash == test {
		t.Fatalf("expected hash and computed hash are equal %d, %d", test, hash)
	}
}

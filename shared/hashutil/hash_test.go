package hashutil_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestHash(t *testing.T) {
	hashOf0 := [32]byte{47, 163, 246, 134, 223, 135, 105, 149, 22, 126, 124, 46, 93, 116, 196, 199, 182, 228, 143, 128, 104, 254, 14, 68, 32, 131, 68, 212, 128, 247, 144, 76}
	hash := hashutil.Hash([]byte{0})
	if hash != hashOf0 {
		t.Fatalf("expected hash and computed hash are not equal %d, %d", hash, hashOf0)
	}

	hashOf1 := [32]byte{149, 69, 186, 55, 178, 48, 216, 162, 231, 22, 196, 112, 117, 134, 84, 39, 128, 129, 91, 124, 64, 136, 237, 203, 154, 246, 169, 69, 45, 80, 243, 36}
	hash = hashutil.Hash([]byte{1})
	if hash != hashOf1 {
		t.Fatalf("expected hash and computed hash are not equal %d, %d", hash, hashOf1)
	}

	if hashOf0 == hashOf1 {
		t.Fatalf("expected hash and computed hash are equal %d, %d", hash, hashOf1)
	}
}

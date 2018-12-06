package hashutil_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestHash(t *testing.T) {
	hashOf0 := [32]byte{93, 83, 70, 159, 32, 254, 244, 248, 234, 181, 43, 136, 4, 78, 222, 105, 199, 122, 106, 104, 166, 7, 40, 96, 159, 196, 166, 95, 245, 49, 231, 208}
	hash := hashutil.Hash([]byte{0})
	if hash != hashOf0 {
		t.Fatalf("expected hash and computed hash are not equal %d, %d", hash, hashOf0)
	}

	hashOf1 := [32]byte{39, 103, 241, 92, 138, 242, 242, 199, 34, 93, 82, 115, 253, 214, 131, 237, 199, 20, 17, 10, 152, 125, 16, 84, 105, 124, 52, 138, 237, 78, 108, 199}
	hash = hashutil.Hash([]byte{1})
	if hash != hashOf1 {
		t.Fatalf("expected hash and computed hash are not equal %d, %d", hash, hashOf1)
	}

	if hashOf0 == hashOf1 {
		t.Fatalf("expected hash and computed hash are equal %d, %d", hash, hashOf1)
	}
}

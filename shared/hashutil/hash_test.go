package hashutil_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestHash(t *testing.T) {
	hashOf0 := [32]byte{188, 54, 120, 158, 122, 30, 40, 20, 54, 70, 66, 41, 130, 143, 129, 125, 102, 18, 247, 180, 119, 214, 101, 145, 255, 150, 169, 224, 100, 188, 201, 138}
	hash := hashutil.Hash([]byte{0})
	if hash != hashOf0 {
		t.Fatalf("expected hash and computed hash are not equal %d, %d", hash, hashOf0)
	}

	hashOf1 := [32]byte{95, 231, 249, 119, 231, 29, 186, 46, 161, 166, 142, 33, 5, 123, 238, 187, 155, 226, 172, 48, 198, 65, 10, 163, 141, 79, 63, 190, 65, 220, 255, 210}
	hash = hashutil.Hash([]byte{1})
	if hash != hashOf1 {
		t.Fatalf("expected hash and computed hash are not equal %d, %d", hash, hashOf1)
	}

	if hashOf0 == hashOf1 {
		t.Fatalf("expected hash and computed hash are equal %d, %d", hash, hashOf1)
	}
}

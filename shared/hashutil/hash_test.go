package hashutil_test

import (
	"encoding/hex"
	"testing"

	bytesutil "github.com/prysmaticlabs/prysm/shared/bytes"
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

	// Same hashing test from go-ethereum for keccak256
	hashOfabc, _ := hex.DecodeString("4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45")
	hash = hashutil.Hash([]byte("abc"))

	h := bytesutil.ToBytes32(hashOfabc)

	if hash != h {
		t.Fatalf("expected hash and computed hash are not equal %d, %d", hash, h)
	}

	if hashOf0 == hashOf1 {
		t.Fatalf("expected hash and computed hash are equal %d, %d", hash, hashOf1)
	}
}

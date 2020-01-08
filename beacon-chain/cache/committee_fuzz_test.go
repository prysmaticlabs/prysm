package cache

import (
	"testing"

	fuzz "github.com/google/gofuzz"
)

func TestCommitteeKeyFuzz_OK(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	c := &Committees{}
	fuzzer.Funcs(c)

	k, err := committeeKeyFn(c)
	if err != nil {
		t.Fatal(err)
	}
	if k != key(c.Seed) {
		t.Errorf("Incorrect hash k: %s, expected %s", k, key(c.Seed))
	}
}
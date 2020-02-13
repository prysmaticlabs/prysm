package kv

import (
	"testing"

	testpb "github.com/prysmaticlabs/prysm/proto/testing"
)

func foo() *testpb.Puzzle {
	return nil
}

func Test_encode_handlesNilFromFunction(t *testing.T) {
	_, err := encode(foo())
	if err == nil || err.Error() != "cannot encode nil message" {
		t.Fatalf("Wrong error %v", err)
	}
}
package kv

import (
	"context"
	"testing"

	testpb "github.com/prysmaticlabs/prysm/proto/testing"
)

func Test_encode_handlesNilFromFunction(t *testing.T) {
	foo := func() *testpb.Puzzle {
		return nil
	}
	_, err := encode(context.Background(), foo())
	if err == nil || err.Error() != "cannot encode nil message" {
		t.Fatalf("Wrong error %v", err)
	}
}

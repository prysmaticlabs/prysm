package sync_test

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/testing"
)

func TestSubscribe(t *testing.T) {
	_ = &pb.TestSimpleMessage{}
	t.Fail()
}

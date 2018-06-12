package p2p

import (
	"testing"

	"github.com/ethereum/go-ethereum/sharding"
)

// Verifies that Server implements the ShardP2P interface.
var _ = sharding.ShardP2P(&Server{})

func TestFeed_ReturnsSameFeed(t *testing.T) {
	tests := []struct {
		a interface{}
		b interface{}
	}{
		{a: 1, b: 2},
		{a: 'a', b: 'b'},
		{a: struct{ c int }{c: 1}, b: struct{ c int }{c: 2}},
		{a: struct{ c string }{c: "a"}, b: struct{ c string }{c: "b"}},
	}

	s, _ := NewServer()

	for _, tt := range tests {
		feed1, _ := s.Feed(tt.a)
		feed2, _ := s.Feed(tt.b)

		if feed1 != feed2 {
			t.Errorf("Expected %v to be equal to %v", feed1, feed2)
		}
	}
}

package protoarray

import (
	"context"
	"testing"
)

func TestStore_leadsToViableHead(t *testing.T) {
	tests := []struct {
		n              *Node
		justifiedEpoch uint64
		finalizedEpoch uint64
		want           bool
	}{
		{&Node{}, 0, 0, true},
		{&Node{}, 1, 0, false},
		{&Node{}, 0, 1, false},
		{&Node{finalizedEpoch: 1, justifiedEpoch: 1}, 1, 1, true},
		{&Node{finalizedEpoch: 1, justifiedEpoch: 1}, 2, 2, false},
		{&Node{finalizedEpoch: 3, justifiedEpoch: 4}, 4, 3, true},
	}
	for _, tc := range tests {
		s := &Store{
			justifiedEpoch: tc.justifiedEpoch,
			finalizedEpoch: tc.finalizedEpoch,
			nodes:          []*Node{tc.n},
		}
		got, err := s.leadsToViableHead(context.Background(), tc.n)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("viableForHead() = %v, want %v", got, tc.want)
		}
	}
}

func TestStore_viableForHead(t *testing.T) {
	tests := []struct {
		n              *Node
		justifiedEpoch uint64
		finalizedEpoch uint64
		want           bool
	}{
		{&Node{}, 0, 0, true},
		{&Node{}, 1, 0, false},
		{&Node{}, 0, 1, false},
		{&Node{finalizedEpoch: 1, justifiedEpoch: 1}, 1, 1, true},
		{&Node{finalizedEpoch: 1, justifiedEpoch: 1}, 2, 2, false},
		{&Node{finalizedEpoch: 3, justifiedEpoch: 4}, 4, 3, true},
	}
	for _, tc := range tests {
		s := &Store{
			justifiedEpoch: tc.justifiedEpoch,
			finalizedEpoch: tc.finalizedEpoch,
		}
		if got := s.viableForHead(context.Background(), tc.n); got != tc.want {
			t.Errorf("viableForHead() = %v, want %v", got, tc.want)
		}
	}
}

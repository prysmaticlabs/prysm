package p2p

import "testing"

func TestFeed_ConcurrentWrite(t *testing.T) {
	s, err := NewServer()
	if err != nil {
		t.Fatalf("could not create server %v", err)
	}

	for i := 0; i < 5; i++ {
		go s.Feed("a")
	}
}

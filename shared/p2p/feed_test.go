package p2p

import (
	"reflect"
	"sync"
	"testing"

	"github.com/gogo/protobuf/proto"
	testpb "github.com/prysmaticlabs/prysm/proto/testing"
)

func TestFeed_SameFeed(t *testing.T) {
	tests := []struct {
		a    proto.Message
		b    proto.Message
		want bool
	}{
		// Equality tests
		{a: &testpb.TestMessage{}, b: &testpb.TestMessage{}, want: true},
		{a: &testpb.Puzzle{}, b: &testpb.Puzzle{}, want: true},
		// Inequality tests
		{a: &testpb.TestMessage{}, b: &testpb.Puzzle{}, want: false},
		{a: &testpb.Puzzle{}, b: &testpb.TestMessage{}, want: false},
	}

	s, _ := NewServer(&ServerConfig{})

	for _, tt := range tests {
		feed1 := s.Feed(tt.a)
		feed2 := s.Feed(tt.b)

		if (feed1 == feed2) != tt.want {
			t.Errorf("Expected %v == %v to be %t", feed1, feed2, tt.want)
		}
	}
}

func TestFeed_ConcurrentWrite(t *testing.T) {
	s := Server{
		feeds: make(map[reflect.Type]Feed),
		mutex: &sync.Mutex{},
	}

	for i := 0; i < 5; i++ {
		go s.Feed(&testpb.TestMessage{})
	}
}

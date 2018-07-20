package p2p

import (
	"reflect"
	"testing"
)

func TestFeed_ReturnsSameFeed(t *testing.T) {
	tests := []struct {
		a    interface{}
		b    interface{}
		want bool
	}{
		// Equality tests
		{a: 1, b: 2, want: true},
		{a: 'a', b: 'b', want: true},
		{a: struct{ c int }{c: 1}, b: struct{ c int }{c: 2}, want: true},
		{a: struct{ c string }{c: "a"}, b: struct{ c string }{c: "b"}, want: true},
		{a: reflect.TypeOf(struct{ c int }{c: 1}), b: struct{ c int }{c: 2}, want: true},
		// Inequality tests
		{a: 1, b: '2', want: false},
		{a: 'a', b: 1, want: false},
		{a: struct{ c int }{c: 1}, b: struct{ c int64 }{c: 2}, want: false},
		{a: struct{ c string }{c: "a"}, b: struct{ c float64 }{c: 3.4}, want: false},
	}

	s, _ := NewServer()

	for _, tt := range tests {
		feed1 := s.Feed(tt.a)
		feed2 := s.Feed(tt.b)

		if (feed1 == feed2) != tt.want {
			t.Errorf("Expected %v == %v to be %t", feed1, feed2, tt.want)
		}
	}
}

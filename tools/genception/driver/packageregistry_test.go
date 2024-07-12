package driver

import (
	"strings"
	"testing"
)

func TestIsSuperset(t *testing.T) {
	cases := []struct {
		a        []string
		b        []string
		expected bool
	}{
		{[]string{"a", "b", "c", "d"}, []string{"a", "b"}, true},
		{[]string{"a", "b", "c", "d"}, []string{"a", "b", "c", "d"}, true},
		{[]string{"a", "b", "c", "d"}, []string{"a", "b", "c", "d", "e"}, false},
		{[]string{"a", "b", "c", "d"}, []string{"a", "b", "c"}, true},
		{[]string{}, []string{"a"}, false},
	}
	for _, c := range cases {
		t.Run(strings.Join(c.a, "_")+"__"+strings.Join(c.b, "_"), func(t *testing.T) {
			if isSuperset(c.a, c.b) != c.expected {
				t.Errorf("isSuperset(%v, %v) != %v", c.a, c.b, c.expected)
			}
		})
	}
}

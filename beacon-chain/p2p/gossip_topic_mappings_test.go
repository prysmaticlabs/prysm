package p2p

import (
	"reflect"
	"testing"
)

func TestMappingHasNoDuplicates(t *testing.T) {
	m := make(map[reflect.Type]bool)
	for _, v := range GossipTopicMappings {
		if _, ok := m[reflect.TypeOf(v)]; ok {
			t.Errorf("%T is duplicated in the topic mapping", v)
		}
		m[reflect.TypeOf(v)] = true
	}
}

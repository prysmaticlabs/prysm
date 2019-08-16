package filters

import (
	"testing"
)

func TestQueryFilter_ChainsCorrectly(t *testing.T) {
	f := NewFilter().
		SetStartSlot(2).
		SetEndSlot(4).
		SetParentRoot([]byte{3, 4, 5}).
		SetShard(0)
	filterSet := f.Filters()
	if len(filterSet) != 4 {
		t.Errorf("Expected 4 filters to have been set, received %d", len(filterSet))
	}
	for k, v := range filterSet {
		switch k {
		case StartSlot:
			t.Log(v.(uint64))
		case EndSlot:
			t.Log(v.(uint64))
		case ParentRoot:
			t.Log(v.([]byte))
		case Shard:
			t.Log(v.(uint64))
		default:
			t.Log("Unknown filter type")
		}
	}
}

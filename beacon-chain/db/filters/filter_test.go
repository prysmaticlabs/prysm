package filters

import (
	"testing"
)

func TestQueryFilter_ChainsCorrectly(t *testing.T) {
	f := NewFilter().
		SetStartSlot(2).
		SetEndSlot(4).
		SetParentRoot([32]byte{3, 4, 5}).
		SetRoot([32]byte{}).
		SetShard(0)
	filterSet := f.Filters()
	if len(filterSet) != 5 {
		t.Errorf("Expected 5 filters to have been set, received %d", len(filterSet))
	}
	for k, v := range filterSet {
		switch k {
		case StartSlot:
			t.Log(v.(uint64))
		case EndSlot:
			t.Log(v.(uint64))
		case ParentRoot:
			t.Log(v.([32]byte))
		case Shard:
			t.Log(v.(uint64))
		default:
			t.Log("Unknown filter type")
		}
	}
}

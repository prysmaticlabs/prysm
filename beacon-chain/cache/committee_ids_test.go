package cache

import (
	"reflect"
	"testing"
)

func TestCommitteeIDCache_RoundTrip(t *testing.T) {
	c := newCommitteeIDs()
	slot := uint64(100)
	committeeIDs := c.GetAggregatorCommitteeIDs(slot)
	if len(committeeIDs) != 0 {
		t.Errorf("Empty cache returned an object: %v", committeeIDs)
	}

	c.AddAggregatorCommiteeID(slot, 1)
	res := c.GetAggregatorCommitteeIDs(slot)
	if !reflect.DeepEqual(res, []uint64{1}) {
		t.Error("Expected equal value to return from cache")
	}

	c.AddAggregatorCommiteeID(slot, 2)
	res = c.GetAggregatorCommitteeIDs(slot)
	if !reflect.DeepEqual(res, []uint64{1, 2}) {
		t.Error("Expected equal value to return from cache")
	}

	c.AddAggregatorCommiteeID(slot, 3)
	res = c.GetAggregatorCommitteeIDs(slot)
	if !reflect.DeepEqual(res, []uint64{1, 2, 3}) {
		t.Error("Expected equal value to return from cache")
	}

	committeeIDs = c.GetAttesterCommitteeIDs(slot)
	if len(committeeIDs) != 0 {
		t.Errorf("Empty cache returned an object: %v", committeeIDs)
	}

	c.AddAttesterCommiteeID(slot, 11)
	res = c.GetAttesterCommitteeIDs(slot)
	if !reflect.DeepEqual(res, []uint64{11}) {
		t.Error("Expected equal value to return from cache")
	}

	c.AddAttesterCommiteeID(slot, 22)
	res = c.GetAttesterCommitteeIDs(slot)
	if !reflect.DeepEqual(res, []uint64{11, 22}) {
		t.Error("Expected equal value to return from cache")
	}

	c.AddAttesterCommiteeID(slot, 33)
	res = c.GetAttesterCommitteeIDs(slot)
	if !reflect.DeepEqual(res, []uint64{11, 22, 33}) {
		t.Error("Expected equal value to return from cache")
	}
}

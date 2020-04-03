package cache

import (
	"reflect"
	"testing"
)

func TestCommitteeIDCache_RoundTrip(t *testing.T) {
	c := newCommitteeIDs()
	slot := uint64(100)
	committeeIDs := c.GetIDs(slot)
	if len(committeeIDs) != 0 {
		t.Errorf("Empty cache returned an object: %v", committeeIDs)
	}

	c.AddID(slot, 1)
	res := c.GetIDs(slot)
	if !reflect.DeepEqual(res, []uint64{1}) {
		t.Error("Expected equal value to return from cache")
	}

	c.AddID(slot, 2)
	res = c.GetIDs(slot)
	if !reflect.DeepEqual(res, []uint64{1, 2}) {
		t.Error("Expected equal value to return from cache")
	}

	c.AddID(slot, 3)
	res = c.GetIDs(slot)
	if !reflect.DeepEqual(res, []uint64{1, 2, 3}) {
		t.Error("Expected equal value to return from cache")
	}
}

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

func TestCommitteeIDs_PersistentCommitteeRoundtrip(t *testing.T) {
	pubkeySet := [][48]byte{}
	c := newCommitteeIDs()

	for i := 0; i < 20; i++ {
		pubkey := [48]byte{byte(i)}
		pubkeySet = append(pubkeySet, pubkey)
		c.AddPersistentCommittee(pubkey[:], []uint64{uint64(i)}, 0)
	}

	for i := 0; i < 20; i++ {
		pubkey := [48]byte{byte(i)}

		idxs, ok, _ := c.GetPersistentCommittees(pubkey[:])
		if !ok {
			t.Errorf("Couldn't find entry in cache for pubkey %#x", pubkey)
			continue
		}
		if int(idxs[0]) != i {
			t.Fatalf("Wanted index of %d but got %d", i, idxs[0])
		}
	}
	coms := c.GetAllCommittees()
	if len(coms) != 20 {
		t.Errorf("Number of committees is not %d but is %d", 20, len(coms))
	}
}

package cache

import (
	"reflect"
	"testing"
)

func TestSubnetIDsCache_RoundTrip(t *testing.T) {
	c := newSubnetIDs()
	slot := uint64(100)
	committeeIDs := c.GetAggregatorSubnetIDs(slot)
	if len(committeeIDs) != 0 {
		t.Errorf("Empty cache returned an object: %v", committeeIDs)
	}

	c.AddAggregatorSubnetID(slot, 1)
	res := c.GetAggregatorSubnetIDs(slot)
	if !reflect.DeepEqual(res, []uint64{1}) {
		t.Error("Expected equal value to return from cache")
	}

	c.AddAggregatorSubnetID(slot, 2)
	res = c.GetAggregatorSubnetIDs(slot)
	if !reflect.DeepEqual(res, []uint64{1, 2}) {
		t.Error("Expected equal value to return from cache")
	}

	c.AddAggregatorSubnetID(slot, 3)
	res = c.GetAggregatorSubnetIDs(slot)
	if !reflect.DeepEqual(res, []uint64{1, 2, 3}) {
		t.Error("Expected equal value to return from cache")
	}

	committeeIDs = c.GetAttesterSubnetIDs(slot)
	if len(committeeIDs) != 0 {
		t.Errorf("Empty cache returned an object: %v", committeeIDs)
	}

	c.AddAttesterSubnetID(slot, 11)
	res = c.GetAttesterSubnetIDs(slot)
	if !reflect.DeepEqual(res, []uint64{11}) {
		t.Error("Expected equal value to return from cache")
	}

	c.AddAttesterSubnetID(slot, 22)
	res = c.GetAttesterSubnetIDs(slot)
	if !reflect.DeepEqual(res, []uint64{11, 22}) {
		t.Error("Expected equal value to return from cache")
	}

	c.AddAttesterSubnetID(slot, 33)
	res = c.GetAttesterSubnetIDs(slot)
	if !reflect.DeepEqual(res, []uint64{11, 22, 33}) {
		t.Error("Expected equal value to return from cache")
	}
}

func TestSubnetIDsCache_PersistentCommitteeRoundtrip(t *testing.T) {
	pubkeySet := [][48]byte{}
	c := newSubnetIDs()

	for i := 0; i < 20; i++ {
		pubkey := [48]byte{byte(i)}
		pubkeySet = append(pubkeySet, pubkey)
		c.AddPersistentCommittee(pubkey[:], []uint64{uint64(i)}, 0)
	}

	for i := uint64(0); i < 20; i++ {
		pubkey := [48]byte{byte(i)}

		idxs, ok, _ := c.GetPersistentSubnets(pubkey[:])
		if !ok {
			t.Errorf("Couldn't find entry in cache for pubkey %#x", pubkey)
			continue
		}
		if idxs[0] != i {
			t.Fatalf("Wanted index of %d but got %d", i, idxs[0])
		}
	}
	coms := c.GetAllSubnets()
	if len(coms) != 20 {
		t.Errorf("Number of committees is not %d but is %d", 20, len(coms))
	}
}

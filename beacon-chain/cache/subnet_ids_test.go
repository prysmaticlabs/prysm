package cache

import (
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestSubnetIDsCache_RoundTrip(t *testing.T) {
	c := newSubnetIDs()
	slot := types.Slot(100)
	committeeIDs := c.GetAggregatorSubnetIDs(slot)
	assert.Equal(t, 0, len(committeeIDs), "Empty cache returned an object")

	c.AddAggregatorSubnetID(slot, 1)
	res := c.GetAggregatorSubnetIDs(slot)
	assert.DeepEqual(t, []uint64{1}, res)

	c.AddAggregatorSubnetID(slot, 2)
	res = c.GetAggregatorSubnetIDs(slot)
	assert.DeepEqual(t, []uint64{1, 2}, res)

	c.AddAggregatorSubnetID(slot, 3)
	res = c.GetAggregatorSubnetIDs(slot)
	assert.DeepEqual(t, []uint64{1, 2, 3}, res)

	committeeIDs = c.GetAttesterSubnetIDs(slot)
	assert.Equal(t, 0, len(committeeIDs), "Empty cache returned an object")

	c.AddAttesterSubnetID(slot, 11)
	res = c.GetAttesterSubnetIDs(slot)
	assert.DeepEqual(t, []uint64{11}, res)

	c.AddAttesterSubnetID(slot, 22)
	res = c.GetAttesterSubnetIDs(slot)
	assert.DeepEqual(t, []uint64{11, 22}, res)

	c.AddAttesterSubnetID(slot, 33)
	res = c.GetAttesterSubnetIDs(slot)
	assert.DeepEqual(t, []uint64{11, 22, 33}, res)
}

func TestSubnetIDsCache_PersistentCommitteeRoundtrip(t *testing.T) {
	c := newSubnetIDs()

	for i := 0; i < 20; i++ {
		pubkey := [fieldparams.BLSPubkeyLength]byte{byte(i)}
		c.AddPersistentCommittee(pubkey[:], []uint64{uint64(i)}, 0)
	}

	for i := uint64(0); i < 20; i++ {
		pubkey := [fieldparams.BLSPubkeyLength]byte{byte(i)}

		idxs, ok, _ := c.GetPersistentSubnets(pubkey[:])
		if !ok {
			t.Errorf("Couldn't find entry in cache for pubkey %#x", pubkey)
			continue
		}
		require.Equal(t, i, idxs[0])
	}
	coms := c.GetAllSubnets()
	assert.Equal(t, 20, len(coms))
}

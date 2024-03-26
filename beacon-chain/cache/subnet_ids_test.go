package cache

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
)

func TestSubnetIDsCache_RoundTrip(t *testing.T) {
	c, err := newSubnetIDs()
	assert.NoError(t, err)

	slot := primitives.Slot(100)
	committeeIDs, err := c.GetAggregatorSubnetIDs(slot)
	assert.Equal(t, err, nil)
	assert.Equal(t, 0, len(committeeIDs), "Empty cache returned an object")

	err = c.AddAggregatorSubnetID(slot, 1)
	assert.NoError(t, err)
	res, err := c.GetAggregatorSubnetIDs(slot)
	assert.NoError(t, err)
	assert.DeepEqual(t, []uint64{1}, res)

	err = c.AddAggregatorSubnetID(slot, 2)
	assert.NoError(t, err)
	res, err = c.GetAggregatorSubnetIDs(slot)
	assert.NoError(t, err)
	assert.DeepEqual(t, []uint64{1, 2}, res)

	err = c.AddAggregatorSubnetID(slot, 3)
	assert.NoError(t, err)
	res, err = c.GetAggregatorSubnetIDs(slot)
	assert.NoError(t, err)
	assert.DeepEqual(t, []uint64{1, 2, 3}, res)

	committeeIDs, err = c.GetAttesterSubnetIDs(slot)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(committeeIDs), "Empty cache returned an object")

	err = c.AddAttesterSubnetID(slot, 11)
	assert.NoError(t, err)
	res, err = c.GetAttesterSubnetIDs(slot)
	assert.NoError(t, err)
	assert.DeepEqual(t, []uint64{11}, res)

	err = c.AddAttesterSubnetID(slot, 22)
	assert.NoError(t, err)
	res, err = c.GetAttesterSubnetIDs(slot)
	assert.NoError(t, err)
	assert.DeepEqual(t, []uint64{11, 22}, res)

	err = c.AddAttesterSubnetID(slot, 33)
	assert.NoError(t, err)
	res, err = c.GetAttesterSubnetIDs(slot)
	assert.NoError(t, err)
	assert.DeepEqual(t, []uint64{11, 22, 33}, res)
}

func TestSubnetIDsCache_PersistentCommitteeRoundtrip(t *testing.T) {
	c, err := newSubnetIDs()
	assert.NoError(t, err)

	c.AddPersistentCommittee([]uint64{0, 1, 2, 7, 8}, 0)

	coms := c.GetAllSubnets()
	assert.Equal(t, 5, len(coms))
}

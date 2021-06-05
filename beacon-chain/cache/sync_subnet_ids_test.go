package cache

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSyncSubnetIDsCache_Roundtrip(t *testing.T) {
	c := newSyncSubnetIDs()

	for i := 0; i < 20; i++ {
		pubkey := [48]byte{byte(i)}
		c.AddSyncCommitteeSubnets(pubkey[:], []uint64{uint64(i)}, 0)
	}

	for i := uint64(0); i < 20; i++ {
		pubkey := [48]byte{byte(i)}

		idxs, ok, _ := c.GetSyncCommitteeSubnets(pubkey[:])
		if !ok {
			t.Errorf("Couldn't find entry in cache for pubkey %#x", pubkey)
			continue
		}
		require.Equal(t, i, idxs[0])
	}
	coms := c.GetAllSubnets()
	assert.Equal(t, 20, len(coms))
}

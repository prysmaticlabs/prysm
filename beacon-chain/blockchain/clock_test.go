package blockchain

import (
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/testing/require"
	"testing"
	"time"
)

func TestClock_GenesisTime(t *testing.T) {
	n := time.Now()
	gt := NewClock(n)
	gtt := gt.GenesisTime()
	require.Equal(t, gt.Time, gtt)
	require.Equal(t, n, gtt)
}

func TestWithNow(t *testing.T) {
	genUnix := time.Unix(0, 0)
	var expectedSlots uint64 = 7200 // a day worth of slots
	now := genUnix.Add(time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot * expectedSlots))
	fn := func() time.Time {
		return now
	}

	gt := NewClock(genUnix, WithNow(fn))
	// in this scenario, "genesis" is exactly 24 hours before "now"
	// so "now" should be 7200 slots after "genesis"
	require.Equal(t, types.Slot(expectedSlots), gt.CurrentSlot())
}
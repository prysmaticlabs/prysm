package filesystem

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestSlotByRoot_Summary(t *testing.T) {
	var noneSet, allSet, firstSet, lastSet, oneSet blobIndexMask
	firstSet[0] = true
	lastSet[len(lastSet)-1] = true
	oneSet[1] = true
	for i := range allSet {
		allSet[i] = true
	}
	cases := []struct {
		name     string
		root     [32]byte
		expected *blobIndexMask
	}{
		{
			name: "not found",
		},
		{
			name:     "none set",
			expected: &noneSet,
		},
		{
			name:     "index 1 set",
			expected: &oneSet,
		},
		{
			name:     "all set",
			expected: &allSet,
		},
		{
			name:     "first set",
			expected: &firstSet,
		},
		{
			name:     "last set",
			expected: &lastSet,
		},
	}
	sc := newBlobStorageCache()
	for _, c := range cases {
		if c.expected != nil {
			key := rootString(bytesutil.ToBytes32([]byte(c.name)))
			sc.cache[key] = BlobStorageSummary{Slot: 0, mask: *c.expected}
		}
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			key := bytesutil.ToBytes32([]byte(c.name))
			sum := sc.Summary(key)
			for i := range c.expected {
				ui := uint64(i)
				if c.expected == nil {
					require.Equal(t, false, sum.HasIndex(ui))
				} else {
					require.Equal(t, c.expected[i], sum.HasIndex(ui))
				}
			}
		})
	}
}

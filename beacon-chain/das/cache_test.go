package das

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/verification"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestCacheEnsureDelete(t *testing.T) {
	c := newCache()
	require.Equal(t, 0, len(c.entries))
	root := bytesutil.ToBytes32([]byte("root"))
	slot := primitives.Slot(1234)
	k := cacheKey{root: root, slot: slot}
	entry := c.ensure(k)
	require.Equal(t, 1, len(c.entries))
	require.Equal(t, c.entries[k], entry)

	c.delete(k)
	require.Equal(t, 0, len(c.entries))
	var nilEntry *cacheEntry
	require.Equal(t, nilEntry, c.entries[k])
}

func TestDbidxMissing(t *testing.T) {
	cases := []struct {
		name    string
		missing []uint64
		idx     dbidx
		expect  int
	}{
		{
			name:    "all missing",
			missing: []uint64{0, 1, 2, 3, 4, 5},
			idx:     dbidx([fieldparams.MaxBlobsPerBlock]bool{false, false, false, false, false, false}),
			expect:  6,
		},
		{
			name:    "none missing",
			missing: []uint64{},
			idx:     dbidx([fieldparams.MaxBlobsPerBlock]bool{true, true, true, true, true, true}),
			expect:  6,
		},
		{
			name:    "ends missing",
			missing: []uint64{0, 5},
			idx:     dbidx([fieldparams.MaxBlobsPerBlock]bool{false, true, true, true, true, false}),
			expect:  6,
		},
		{
			name:    "middle missing",
			missing: []uint64{1, 2, 3, 4},
			idx:     dbidx([fieldparams.MaxBlobsPerBlock]bool{true, false, false, false, false, true}),
			expect:  6,
		},
		{
			name:    "none expected",
			missing: []uint64{},
			idx:     dbidx([fieldparams.MaxBlobsPerBlock]bool{false, false, false, false, false, false}),
			expect:  0,
		},
		{
			name:    "middle missing, half expected",
			missing: []uint64{1, 2},
			idx:     dbidx([fieldparams.MaxBlobsPerBlock]bool{true, false, false, false, false, true}),
			expect:  3,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := c.idx.missing(c.expect)
			require.DeepEqual(t, m, c.missing)
		})
	}
}

func commitsForSidecars(scs []*ethpb.BlobSidecar) [][]byte {
	m := make([][]byte, len(scs))
	for i := range scs {
		m[i] = scs[i].KzgCommitment
	}
	return m
}

func generateMinimalBlobSidecars(t *testing.T, n int) ([]*ethpb.BlobSidecar, []blocks.VerifiedROBlob) {
	scs := make([]*ethpb.BlobSidecar, n)
	vscs := make([]blocks.VerifiedROBlob, n)
	for i := 0; i < n; i++ {
		scs[i] = &ethpb.BlobSidecar{
			Index:         uint64(i),
			KzgCommitment: bytesutil.PadTo([]byte{byte(i)}, 48),
		}
		rob, err := blocks.NewROBlob(scs[i])
		require.NoError(t, err)
		vscs[i] = verification.FakeVerifyForTest(t, rob)
	}
	return scs, vscs
}

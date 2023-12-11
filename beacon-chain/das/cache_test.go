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

func TestNewEntry(t *testing.T) {
	entry := &cacheEntry{}
	require.Equal(t, false, entry.dbidxInitialized())
	entry.ensureDbidx(entry.dbx)
	require.Equal(t, true, entry.dbidxInitialized())
}

func TestDbidxMissing(t *testing.T) {
	cases := []struct {
		name     string
		nMissing int
	}{
		{
			name:     "all missing",
			nMissing: fieldparams.MaxBlobsPerBlock,
		},
		{
			name:     "none missing",
			nMissing: 0,
		},
		{
			name:     "2 missing",
			nMissing: 2,
		},
		{
			name:     "3 missing",
			nMissing: 3,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var idx [fieldparams.MaxBlobsPerBlock]bool
			missing := make([]uint64, 0)
			for i := range idx {
				if i < c.nMissing {
					idx[i] = false
					missing = append(missing, uint64(i))
				} else {
					idx[i] = true
				}
			}
			entry := &cacheEntry{}
			d := entry.ensureDbidx(idx)
			m := d.missing(6)
			require.DeepEqual(t, m, missing)
			require.Equal(t, c.nMissing, len(m))
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

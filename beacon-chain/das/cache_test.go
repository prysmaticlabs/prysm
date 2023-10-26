package das

import (
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
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
	entry.ensureDbidx()
	require.Equal(t, true, entry.dbidxInitialized())
}

func TestDbidxBounds(t *testing.T) {
	scs := generateMinimalBlobSidecars(2)
	entry := &cacheEntry{}
	entry.ensureDbidx(scs...)
	//require.Equal(t, 2, len(entry.dbidx()))
	for i := range scs {
		require.Equal(t, bytesutil.ToBytes48(scs[i].KzgCommitment), *entry.dbidx()[i])
	}

	var nilPtr *[48]byte
	// test that duplicate sidecars are ignored
	orig := entry.dbidx()
	copy(scs[0].KzgCommitment[0:4], []byte("derp"))
	edited := bytesutil.ToBytes48(scs[0].KzgCommitment)
	require.Equal(t, false, *entry.dbidx()[0] == edited)
	entry.ensureDbidx(scs[0])
	for i := 2; i < fieldparams.MaxBlobsPerBlock; i++ {
		require.Equal(t, entry.dbidx()[i], nilPtr)
	}
	require.Equal(t, entry.dbidx(), orig)

	// test that excess sidecars are discarded
	oob := generateMinimalBlobSidecars(fieldparams.MaxBlobsPerBlock + 1)
	entry = &cacheEntry{}
	entry.ensureDbidx(oob...)
	require.Equal(t, fieldparams.MaxBlobsPerBlock, len(entry.dbidx()))
}

func generateMinimalBlobSidecars(n int) []*ethpb.BlobSidecar {
	scs := make([]*ethpb.BlobSidecar, n)
	for i := 0; i < n; i++ {
		scs[i] = &ethpb.BlobSidecar{
			Index:         uint64(i),
			KzgCommitment: bytesutil.PadTo([]byte{byte(i)}, 48),
		}
	}
	return scs
}

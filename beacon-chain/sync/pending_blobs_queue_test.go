package sync

import (
	"context"
	"testing"
	"time"

	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	dbtest "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/doubly-linked-tree"
	p2ptest "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/v4/beacon-chain/sync/initial-sync/testing"
	lruwrpr "github.com/prysmaticlabs/prysm/v4/cache/lru"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestProcessBlobsFromSidecars(t *testing.T) {
	db := dbtest.SetupDB(t)
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	chainService := &mock.ChainService{Genesis: time.Now(), FinalizedCheckPoint: &eth.Checkpoint{}, DB: db}
	stateGen := stategen.New(db, doublylinkedtree.New())
	s := &Service{
		pendingBlobSidecars: newPendingBlobSidecars(),
		seenBlobCache:       lruwrpr.New(10),
		cfg: &config{
			p2p:         p,
			initialSync: &mockSync.Sync{},
			chain:       chainService,
			stateGen:    stateGen,
			clock:       startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot)}}

	b := util.NewBlobsidecar()
	b.Message.Slot = chainService.CurrentSlot() + 1
	beaconState, privKeys := util.DeterministicGenesisState(t, 100)

	bb := util.NewBeaconBlock()
	signedBb, err := blocks.NewSignedBeaconBlock(bb)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, signedBb))
	r, err := signedBb.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, r))

	b.Message.BlockParentRoot = r[:]
	b.Message.ProposerIndex = 21
	b.Signature, err = signing.ComputeDomainAndSign(beaconState, 0, b.Message, params.BeaconConfig().DomainBlobSidecar, privKeys[21])
	require.NoError(t, err)

	s.pendingBlobSidecars.add(b)
	require.Equal(t, 1, len(s.pendingBlobSidecars.blobSidecars))
	s.processBlobsFromSidecars(ctx, r)
	require.Equal(t, 0, len(s.pendingBlobSidecars.blobSidecars))

	// Make sure chain service has the blob.
	require.DeepEqual(t, chainService.Blobs[0], b.Message)
}

func TestPendingBlobSidecars(t *testing.T) {
	// Test Initialization
	cache := newPendingBlobSidecars()
	require.Equal(t, 0, len(cache.blobSidecars))

	// Test Add
	parentRoot := [32]byte{1, 2, 3}
	blob := &eth.SignedBlobSidecar{Message: &eth.BlobSidecar{}}
	blob.Message.BlockParentRoot = parentRoot[:]
	cache.add(blob)
	_, exists := cache.blobSidecars[parentRoot]
	require.Equal(t, true, exists)
	require.Equal(t, 1, len(cache.blobSidecars))

	// Test Add duplicates
	cache.add(blob)
	require.Equal(t, 1, len(cache.blobSidecars))

	// Test Pop
	poppedBlob := cache.pop(parentRoot)
	require.Equal(t, 0, len(cache.blobSidecars))
	require.DeepEqual(t, poppedBlob[0].Message.BlockParentRoot, parentRoot[:])

	// Test Cleanup
	// For this, we can manually set an expired time to simulate an expired blob
	cache.add(blob)
	cache.blobSidecars[parentRoot].expiresAt = time.Now().Add(-time.Second)
	cache.cleanup()
	_, exists = cache.blobSidecars[parentRoot]
	require.Equal(t, false, exists)
}

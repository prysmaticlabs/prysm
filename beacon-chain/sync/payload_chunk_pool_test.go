package sync

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	mock "github.com/OffchainLabs/prysm/v7/beacon-chain/blockchain/testing"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/gloas"
	dbtest "github.com/OffchainLabs/prysm/v7/beacon-chain/db/testing"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/startup"
	lruwrpr "github.com/OffchainLabs/prysm/v7/cache/lru"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func newChunkTestService(t *testing.T, blockRoot [32]byte) *Service {
	t.Helper()
	db := dbtest.SetupDB(t)
	// Genesis one slot ago so the wall clock sits in slot 1.
	genesis := time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0)
	chain := &mock.ChainService{
		Genesis:            genesis,
		DB:                 db,
		InitSyncBlockRoots: map[[32]byte]bool{blockRoot: true},
	}
	return &Service{
		ctx:                      context.Background(),
		chainStarted:             &atomic.Bool{},
		pendingPayloadEnvelopes:  make(map[[32]byte]map[uint64]*ethpb.SignedExecutionPayloadEnvelope),
		pendingPayloadChunks:     make(map[[32]byte]*pendingChunkEntry),
		payloadChunkPool:         newPayloadChunkPool(),
		payloadChunkCommitments:  newSlotAwareCache(16),
		seenPayloadChunkCache:    newSlotAwareCache(1024),
		seenPayloadEnvelopeCache: lruwrpr.New(10),
		badBlockCache:            lruwrpr.New(10),
		cfg: &config{
			chain:    chain,
			beaconDB: db,
			clock:    startup.NewClock(genesis, chain.ValidatorsRoot),
		},
	}
}

// A chunk set that completes before its block binding is known must not be
// lost or permanently misrouted: the unauthenticated beacon_block_root hint is
// only a hint, and the authoritative binding retries the import.
func TestReconstructedEnvelopeBindingRetry(t *testing.T) {
	const slot = primitives.Slot(1)
	builderIdx := primitives.BuilderIndex(7)
	blockRoot := [32]byte{0xAB, 0xCD}
	blockHash := [32]byte{0xBB}
	envelope := testSignedExecutionPayloadEnvelope(t, slot, builderIdx, [32]byte{}, blockHash).Message

	waitSeen := func(t *testing.T, s *Service) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if s.hasSeenPayloadEnvelope(blockRoot, builderIdx) {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("timed out waiting for the reconstructed envelope import")
	}

	t.Run("zero hint waits for the binding", func(t *testing.T) {
		s := newChunkTestService(t, blockRoot)
		chunks, chunksRoot, err := gloas.BuildPayloadChunkMessages(envelope, slot, [32]byte{})
		require.NoError(t, err)

		// The bid was seen on gossip only: commitment known, binding unknown.
		s.registerPayloadChunkCommitment(slot, chunksRoot, [32]byte{})
		for _, c := range chunks {
			s.acceptVerifiedPayloadChunk(s.ctx, c)
		}
		// Complete, but with no binding and no hint nothing can be imported yet.
		_, done := s.payloadChunkPool.completedChunks(chunksRoot, slot)
		require.Equal(t, true, done)
		require.Equal(t, false, s.hasSeenPayloadEnvelope(blockRoot, builderIdx))

		// The block arrives: the binding registers and the retry imports the set.
		s.registerPayloadChunkCommitment(slot, chunksRoot, blockRoot)
		waitSeen(t, s)
	})

	t.Run("forged hint is corrected by the binding", func(t *testing.T) {
		s := newChunkTestService(t, blockRoot)
		forged := [32]byte{0xFF, 0xEE}
		chunks, chunksRoot, err := gloas.BuildPayloadChunkMessages(envelope, slot, forged)
		require.NoError(t, err)

		s.registerPayloadChunkCommitment(slot, chunksRoot, [32]byte{})
		for _, c := range chunks {
			s.acceptVerifiedPayloadChunk(s.ctx, c)
		}
		// Misrouted: parked under the forged root, nothing imported, and the
		// pool entry is not marked imported (hint routing is not authoritative).
		require.Equal(t, false, s.hasSeenPayloadEnvelope(blockRoot, builderIdx))
		s.pendingEnvelopeLock.Lock()
		_, parked := s.pendingPayloadEnvelopes[forged]
		s.pendingEnvelopeLock.Unlock()
		require.Equal(t, true, parked)
		_, done := s.payloadChunkPool.completedChunks(chunksRoot, slot)
		require.Equal(t, true, done)

		s.registerPayloadChunkCommitment(slot, chunksRoot, blockRoot)
		waitSeen(t, s)
	})

	t.Run("authoritative routing is not retried", func(t *testing.T) {
		s := newChunkTestService(t, blockRoot)
		chunks, chunksRoot, err := gloas.BuildPayloadChunkMessages(envelope, slot, blockRoot)
		require.NoError(t, err)

		// Binding known before completion: import happens inline and the pool
		// entry is consumed for retry purposes.
		s.registerPayloadChunkCommitment(slot, chunksRoot, blockRoot)
		for _, c := range chunks {
			s.acceptVerifiedPayloadChunk(s.ctx, c)
		}
		require.Equal(t, true, s.hasSeenPayloadEnvelope(blockRoot, builderIdx))
		_, done := s.payloadChunkPool.completedChunks(chunksRoot, slot)
		require.Equal(t, false, done)
	})
}

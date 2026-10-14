package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/blockchain"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/gloas"
	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
)

// maxChunkPoolRoots bounds concurrent reconstructions; each entry holds at most
// MAX_PAYLOAD_SIZE bytes of chunk data. The commitment cache already gates
// admission, so this is a second-layer bound.
const maxChunkPoolRoots = 8

// chunkPoolEntry accumulates the verified chunks of one commitment.
type chunkPoolEntry struct {
	slot          primitives.Slot
	blockRootHint [32]byte
	chunks        [fieldparams.TotalExecutionPayloadChunks][]byte
	received      int
	firstArrival  time.Time
	completed     bool
	// imported is set once the set was routed using the authoritative block
	// binding. Routing by the unauthenticated hint does not set it, so a
	// forged or missing hint is corrected when the binding registers.
	imported bool
}

// payloadChunkPool buffers verified chunks until all
// TOTAL_EXECUTION_PAYLOAD_CHUNKS are present (no erasure coding: every index is
// required), then hands the set to reconstruction exactly once.
type payloadChunkPool struct {
	mu      sync.Mutex
	entries map[[32]byte]*chunkPoolEntry
}

func newPayloadChunkPool() *payloadChunkPool {
	return &payloadChunkPool{entries: make(map[[32]byte]*chunkPoolEntry)}
}

// insert adds a verified chunk. It returns the complete ordered chunk set, the
// routing hint, and the time the first chunk arrived exactly once, when the
// final missing index lands.
func (p *payloadChunkPool) insert(c *ethpb.PayloadChunk) (complete bool, chunks [][]byte, hint [32]byte, firstArrival time.Time) {
	chunksRoot := bytesutil.ToBytes32(c.PayloadChunksRoot)

	p.mu.Lock()
	defer p.mu.Unlock()
	entry, ok := p.entries[chunksRoot]
	if !ok {
		if len(p.entries) >= maxChunkPoolRoots {
			log.Debug("Payload chunk pool is full, dropping chunk")
			return false, nil, [32]byte{}, time.Time{}
		}
		entry = &chunkPoolEntry{slot: c.Slot, firstArrival: time.Now()}
		p.entries[chunksRoot] = entry
	}
	if entry.completed || entry.chunks[c.Index] != nil {
		return false, nil, [32]byte{}, time.Time{}
	}
	entry.chunks[c.Index] = c.Chunk
	entry.received++
	if hintRoot := bytesutil.ToBytes32(c.BeaconBlockRoot); entry.blockRootHint == ([32]byte{}) && hintRoot != ([32]byte{}) {
		entry.blockRootHint = hintRoot
	}
	if entry.received < fieldparams.TotalExecutionPayloadChunks {
		return false, nil, [32]byte{}, time.Time{}
	}
	entry.completed = true
	out := make([][]byte, fieldparams.TotalExecutionPayloadChunks)
	copy(out, entry.chunks[:])
	return true, out, entry.blockRootHint, entry.firstArrival
}

// completedChunks returns a copy of a finished chunk set that has not yet been
// routed with an authoritative block binding, for retry once a binding exists.
func (p *payloadChunkPool) completedChunks(chunksRoot [32]byte, slot primitives.Slot) ([][]byte, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry, ok := p.entries[chunksRoot]
	if !ok || !entry.completed || entry.imported || entry.slot != slot {
		return nil, false
	}
	out := make([][]byte, fieldparams.TotalExecutionPayloadChunks)
	copy(out, entry.chunks[:])
	return out, true
}

// markImported records that the set was routed via the authoritative binding,
// stopping further binding-registration retries.
func (p *payloadChunkPool) markImported(chunksRoot [32]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if entry, ok := p.entries[chunksRoot]; ok {
		entry.imported = true
	}
}

// pruneBefore drops entries older than the given slot.
func (p *payloadChunkPool) pruneBefore(slot primitives.Slot) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for root, entry := range p.entries {
		if entry.slot < slot {
			delete(p.entries, root)
		}
	}
}

// acceptVerifiedPayloadChunk buffers a gossip-verified chunk and triggers
// reconstruction when the set completes.
func (s *Service) acceptVerifiedPayloadChunk(ctx context.Context, c *ethpb.PayloadChunk) {
	payloadChunkReceivedTotal.Inc()
	complete, chunks, hint, firstArrival := s.payloadChunkPool.insert(c)
	if !complete {
		return
	}
	payloadChunkReconstructionSeconds.Observe(time.Since(firstArrival).Seconds())
	s.importReconstructedEnvelope(ctx, c.Slot, bytesutil.ToBytes32(c.PayloadChunksRoot), hint, chunks)
}

// importReconstructedEnvelope decodes the completed chunk set and routes the
// envelope into the regular import pipeline: direct import when the owning
// block is known, otherwise the pending-envelope pool (drained on block
// arrival). Decode failure means the builder committed to garbage bytes; the
// chunks were honestly relayed, so there is no one to penalize and the payload
// will simply be voted absent.
func (s *Service) importReconstructedEnvelope(ctx context.Context, slot primitives.Slot, chunksRoot, hint [32]byte, chunks [][]byte) {
	envelope, err := gloas.ReconstructPayloadEnvelope(chunks)
	if err != nil {
		payloadChunkReconstructionTotal.WithLabelValues("decode_failed").Inc()
		log.WithError(err).WithFields(logrus.Fields{
			"slot":       slot,
			"chunksRoot": fmt.Sprintf("%#x", chunksRoot),
		}).Warn("Builder committed to undecodable payload chunk bytes")
		return
	}

	// Prefer the authenticated block binding over the unauthenticated hint. The
	// hint can be forged or zero, so only an authoritative routing marks the
	// pool entry imported; otherwise the set is retried when the binding
	// registers (see drainPendingPayloadChunks).
	root := hint
	authoritative := false
	if bound, known := s.payloadChunkCommitmentBlockRoot(slot, chunksRoot); known && bound != ([32]byte{}) {
		root = bound
		authoritative = true
	}
	if root == ([32]byte{}) {
		payloadChunkReconstructionTotal.WithLabelValues("no_binding").Inc()
		log.Debug("Reconstructed payload envelope is awaiting its beacon block binding")
		return
	}
	if authoritative {
		s.payloadChunkPool.markImported(chunksRoot)
	}
	payloadChunkReconstructionTotal.WithLabelValues("ok").Inc()

	// The zero signature marks the envelope as chunk-authenticated; see
	// gloas.IsChunkAuthenticatedEnvelope.
	signed := &ethpb.SignedExecutionPayloadEnvelope{
		Message:   envelope,
		Signature: make([]byte, fieldparams.BLSSignatureLength),
	}

	if s.cfg.chain.HasBlock(ctx, root) {
		s.receiveReconstructedEnvelope(ctx, signed, root)
		return
	}
	// Block not yet processed: park in the pending-envelope pool, which the
	// block-processed event and the half-slot sweep both drain.
	s.queuePendingPayloadEnvelopeFromRootRequest(signed, root)
}

// receiveReconstructedEnvelope imports a reconstructed envelope, mirroring the
// semantics of the retired envelope gossip subscriber.
func (s *Service) receiveReconstructedEnvelope(ctx context.Context, signed *ethpb.SignedExecutionPayloadEnvelope, root [32]byte) {
	e, err := blocks.WrappedROSignedExecutionPayloadEnvelope(signed, root)
	if err != nil {
		log.WithError(err).Error("Could not wrap reconstructed execution payload envelope")
		return
	}
	env, err := e.Envelope()
	if err != nil {
		log.WithError(err).Error("Could not get reconstructed execution payload envelope")
		return
	}
	if s.hasSeenPayloadEnvelope(root, env.BuilderIndex()) {
		return
	}

	receiveCtx, cancel := context.WithTimeout(ctx, params.BeaconConfig().SlotDuration())
	defer cancel()
	if err := s.cfg.chain.ReceiveExecutionPayloadEnvelope(receiveCtx, e); err != nil {
		if blockchain.IsInvalidBlock(err) {
			s.setBadPayload(ctx, root)
		}
		log.WithError(err).Debug("Could not process reconstructed payload envelope")
		return
	}
	s.setSeenPayloadEnvelope(root, env.BuilderIndex())
	if s.chainIsStarted() {
		go func() {
			if err := s.processPendingBlocks(s.ctx); err != nil {
				log.WithError(err).Debug("Could not process pending blocks after envelope reconstruction")
			}
		}()
	}
}

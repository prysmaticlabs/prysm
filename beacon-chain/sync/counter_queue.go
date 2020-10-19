package sync

import (
	"context"
	"sort"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const costPerSecond = 200

var maxCost = 400 * params.BeaconConfig().SlotsPerEpoch

// Only lookback at attestations/blocks 8 epochs back.
var costThreshold = 8 * params.BeaconConfig().SlotsPerEpoch

type costCounter struct {
	cost uint64
	sync.RWMutex
}

func (c *costCounter) increment() {
	c.Lock()
	if c.cost+costPerSecond > maxCost {
		return
	}
	c.cost += costPerSecond
	c.Unlock()
}

func (c *costCounter) decrement(cost uint64) bool {
	c.Lock()
	defer c.Unlock()
	if c.cost < cost {
		return false
	}
	c.cost -= cost
	return true
}

func (c *costCounter) currentCost() uint64 {
	c.RLock()
	defer c.RUnlock()
	return c.cost
}

type aggregateCostPool struct {
	aggregates []*ethpb.SignedAggregateAttestationAndProof
	sync.RWMutex
}

func (a *aggregateCostPool) addAggregate(agg *ethpb.SignedAggregateAttestationAndProof) {
	a.Lock()
	a.aggregates = append(a.aggregates, agg)
	a.Unlock()
}

func (a *aggregateCostPool) removeAggregate(aggregate *ethpb.SignedAggregateAttestationAndProof) {
	a.Lock()
	defer a.Unlock()
	for i, agg := range a.aggregates {
		// Remove aggregate.
		if ssz.DeepEqual(agg, aggregate) {
			a.aggregates = append(a.aggregates[:i], a.aggregates[i+1:]...)
			break
		}
	}
}

func (a *aggregateCostPool) aggregateSet() []*ethpb.SignedAggregateAttestationAndProof {
	a.RLock()
	defer a.RUnlock()
	return a.aggregates
}

type blockCostPool struct {
	blocks map[[32]byte]*ethpb.SignedBeaconBlock
	sync.RWMutex
}

func (b *blockCostPool) allBlocks() ([]*ethpb.SignedBeaconBlock, [][32]byte) {
	roots := make([][32]byte, 0, len(b.blocks))
	blocks := make([]*ethpb.SignedBeaconBlock, 0, len(b.blocks))
	for key, blk := range b.blocks {
		roots = append(roots, key)
		blocks = append(blocks, blk)
	}
	return blocks, roots
}

func (b *blockCostPool) addBlock(block *ethpb.SignedBeaconBlock) {
	b.Lock()
	defer b.Unlock()
	rt, err := block.Block.HashTreeRoot()
	if err != nil {
		return
	}
	if _, ok := b.blocks[rt]; ok {
		return
	}
	b.blocks[rt] = block
}

func (b *blockCostPool) blockExists(root [32]byte) bool {
	b.RLock()
	defer b.RUnlock()
	_, ok := b.blocks[root]
	return ok
}

func (b *blockCostPool) deleteBlock(block *ethpb.SignedBeaconBlock) {
	b.Lock()
	defer b.Unlock()
	rt, err := block.Block.HashTreeRoot()
	if err != nil {
		return
	}
	delete(b.blocks, rt)
}

func (s *Service) costCounter() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.costCtr.increment()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) costObjectQueue() {
	attTicker := time.NewTicker(10 * time.Second)
	defer attTicker.Stop()
	blockTicker := time.NewTicker(20 * time.Second)
	defer blockTicker.Stop()
	for {
		select {
		case <-blockTicker.C:
			blocks, roots := s.blockCostQueue.allBlocks()
			sort.Slice(blocks, func(i, j int) bool {
				less := blocks[i].Block.Slot < blocks[j].Block.Slot
				if less {
					roots[i], roots[j] = roots[j], roots[i]
				}
				return less
			})
			ch := s.chain.HeadSlot()
			for i, b := range blocks {
				targetSlot, err := s.targetSlotRetriever(s.ctx, bytesutil.ToBytes32(b.Block.ParentRoot))
				if err != nil {
					s.blockCostQueue.deleteBlock(b)
					continue
				}
				if ch < targetSlot {
					s.blockCostQueue.deleteBlock(b)
					continue
				}
				diff := ch - targetSlot
				log.Errorf("processing block with cost %d", diff)
				if !s.costCtr.decrement(diff) {
					break
				}
				if err := s.chain.ReceiveBlock(s.ctx, b, roots[i]); err != nil {
					log.Debugf("Could not process block from slot %d: %v", b.Block.Slot, err)
					s.setBadBlock(s.ctx, roots[i])
					s.blockCostQueue.deleteBlock(b)
					continue
				}
				// Broadcasting the block again once a node is able to process it.
				if err := s.p2p.Broadcast(s.ctx, b); err != nil {
					log.WithError(err).Debug("Failed to broadcast block")
				}
				s.blockCostQueue.deleteBlock(b)
			}
		case <-attTicker.C:
			aggregates := s.attestationCostQueue.aggregateSet()
			if len(aggregates) == 0 {
				continue
			}
			sort.Slice(aggregates, func(i, j int) bool {
				slotI, err := s.targetSlotRetriever(s.ctx, bytesutil.ToBytes32(aggregates[i].Message.Aggregate.Data.Target.Root))
				if err != nil || slotI == 0 {
					return false
				}
				slotJ, err := s.targetSlotRetriever(s.ctx, bytesutil.ToBytes32(aggregates[j].Message.Aggregate.Data.Target.Root))
				// In the event of failure push bad aggregates to the back of the array.
				if err != nil || slotJ == 0 {
					return true
				}
				return slotI < slotJ
			})
			hd := s.chain.HeadSlot()
			log.Errorf("current cost limit: %d", s.costCtr.currentCost())
			// Loop through aggregates.
			for _, ag := range aggregates {
				slotI, err := s.targetSlotRetriever(s.ctx, bytesutil.ToBytes32(ag.Message.Aggregate.Data.Target.Root))
				if err != nil || slotI == 0 {
					s.attestationCostQueue.removeAggregate(ag)
					continue
				}
				if hd < slotI {
					s.attestationCostQueue.removeAggregate(ag)
					continue
				}
				diff := hd - slotI
				log.Errorf("processing aggregate with cost %d", diff)
				if !s.costCtr.decrement(diff) {
					break
				}
				if s.validateAggregatedAtt(s.ctx, ag) == pubsub.ValidationAccept {
					if err := s.attPool.SaveAggregatedAttestation(ag.Message.Aggregate); err != nil {
						s.attestationCostQueue.removeAggregate(ag)
						continue
					}
					// Broadcasting the signed attestation again once a node is able to process it.
					if err := s.p2p.Broadcast(s.ctx, ag.Message); err != nil {
						log.WithError(err).Debug("Failed to broadcast")
					}
				}
				s.attestationCostQueue.removeAggregate(ag)
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) blockCostChecker(ctx context.Context, blk *ethpb.SignedBeaconBlock) (bool, uint64, error) {
	headSlot := s.chain.HeadSlot()
	targetRt := bytesutil.ToBytes32(blk.Block.ParentRoot)
	targetSlot, err := s.targetSlotRetriever(ctx, targetRt)
	if err != nil {
		return false, 0, err
	}
	if headSlot <= targetSlot {
		return true, 0, nil
	}
	diff := headSlot - targetSlot
	if diff > costThreshold {
		return false, diff, nil
	}
	return true, diff, nil
}

func (s *Service) attestationCostChecker(ctx context.Context, att *ethpb.SignedAggregateAttestationAndProof) (bool, uint64, error) {
	targetRt := bytesutil.ToBytes32(att.Message.Aggregate.Data.Target.Root)
	targetSlot, err := s.targetSlotRetriever(ctx, targetRt)
	if err != nil {
		return false, 0, err
	}
	headSlot := s.chain.HeadSlot()
	if headSlot <= targetSlot {
		return true, 0, nil
	}
	diff := headSlot - targetSlot
	if diff > costThreshold {
		return false, diff, nil
	}
	return true, diff, nil
}

func (s *Service) targetSlotRetriever(ctx context.Context, root [32]byte) (uint64, error) {
	var targetSlot uint64
	switch {
	case s.db.HasStateSummary(ctx, root):
		summ, err := s.db.StateSummary(ctx, root)
		if err != nil {
			return 0, err
		}
		if summ == nil {
			return 0, errors.Errorf("target root does not exist in node's chain: %#x", root)
		}
		targetSlot = summ.Slot
	case s.stateSummaryCache.Has(root):
		summ := s.stateSummaryCache.Get(root)
		if summ == nil {
			return 0, errors.Errorf("target root does not exist in node's chain: %#x", root)
		}
		targetSlot = summ.Slot
	case s.db.HasBlock(ctx, root):
		blk, err := s.db.Block(ctx, root)
		if err != nil {
			return 0, err
		}
		if blk == nil {
			return 0, errors.Errorf("target root does not exist in node's chain: %#x", root)
		}
		targetSlot = blk.Block.Slot
	case s.chain.HasInitSyncBlock(root):
		blk := s.chain.GetInitSyncBlock(root)
		if blk == nil {
			return 0, errors.Errorf("target root does not exist in node's chain: %#x", root)
		}
		targetSlot = blk.Block.Slot
	default:
		return 0, errors.Errorf("target root does not exist in node's chain: %#x", root)
	}
	return targetSlot, nil
}

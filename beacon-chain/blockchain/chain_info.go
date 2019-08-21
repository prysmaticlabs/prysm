package blockchain

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// ChainInfoRetriever defines a common interface for methods in blockchain service which
// directly retrieves chain head related data.
type ChainInfoRetriever interface {
	FinalizedCheckpt() *ethpb.Checkpoint
	HeadSlot() uint64
	HeadRoot() []byte
	CanonicalRoot(slot uint64) []byte
}

// FinalizedCheckpt returns the latest finalized checkpoint tracked in fork choice service.
func (c *ChainService) FinalizedCheckpt() *ethpb.Checkpoint {
	return c.forkChoiceStore.FinalizedCheckpt()
}

// HeadSlot returns the slot of the head of the chain.
func (c *ChainService) HeadSlot() uint64 {
	return c.headSlot
}

// HeadRoot returns the root of the head of the chain.
func (c *ChainService) HeadRoot() []byte {
	c.canonicalRootsLock.RLock()
	defer c.canonicalRootsLock.RUnlock()

	return c.canonicalRoots[c.headSlot]
}

// CanonicalRoot returns the canonical root of a given slot.
func (c *ChainService) CanonicalRoot(slot uint64) []byte {
	c.canonicalRootsLock.RLock()
	defer c.canonicalRootsLock.RUnlock()

	return c.canonicalRoots[slot]
}

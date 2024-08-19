package cache

import (
	"bytes"
	"sync"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
)

var (
	// This cache is intended for use by the sync service to store signed execution payload headers after they pass validation.
	// The signed header from this cache could be used by the proposer when proposing the next slot.
	cachedSignedExecutionPayloadHeader = make(map[primitives.Slot][]*enginev1.SignedExecutionPayloadHeader)
	mu                                 sync.RWMutex
)

// SaveSignedExecutionPayloadHeader saves the signed execution payload header to the cache.
// The cache stores headers for up to two slots. If the input slot is higher than the lowest slot
// currently in the cache, the lowest slot is removed to make space for the new header.
// Only the highest value header for a given parent block hash will be stored.
// This function assumes caller already checks header's slot is current or next slot, it doesn't account for slot validation.
func SaveSignedExecutionPayloadHeader(header *enginev1.SignedExecutionPayloadHeader) {
	mu.Lock()
	defer mu.Unlock()

	for s := range cachedSignedExecutionPayloadHeader {
		if s+1 < header.Message.Slot {
			delete(cachedSignedExecutionPayloadHeader, s)
		}
	}

	// Add or update the header in the map
	if _, ok := cachedSignedExecutionPayloadHeader[header.Message.Slot]; !ok {
		cachedSignedExecutionPayloadHeader[header.Message.Slot] = []*enginev1.SignedExecutionPayloadHeader{header}
	} else {
		found := false
		for i, h := range cachedSignedExecutionPayloadHeader[header.Message.Slot] {
			if bytes.Equal(h.Message.ParentBlockHash, header.Message.ParentBlockHash) {
				if header.Message.Value > h.Message.Value {
					cachedSignedExecutionPayloadHeader[header.Message.Slot][i] = header
				}
				found = true
				break
			}
		}
		if !found {
			cachedSignedExecutionPayloadHeader[header.Message.Slot] = append(cachedSignedExecutionPayloadHeader[header.Message.Slot], header)
		}
	}
}

// SignedExecutionPayloadHeader returns the signed payload header for the given slot and parent block hash.
// Returns nil if the header is not found.
func SignedExecutionPayloadHeader(slot primitives.Slot, parentBlockHash []byte) *enginev1.SignedExecutionPayloadHeader {
	mu.RLock()
	defer mu.RUnlock()

	if headers, ok := cachedSignedExecutionPayloadHeader[primitives.Slot(uint64(slot))]; ok {
		for _, header := range headers {
			if bytes.Equal(header.Message.ParentBlockHash, parentBlockHash) {
				return header
			}
		}
	}

	return nil
}

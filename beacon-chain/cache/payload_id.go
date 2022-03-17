package cache

import (
	"sync"

	types "github.com/prysmaticlabs/eth2-types"
)

const AndPayloadIDLength = 30

type FeeRecipientPayloadIDCache struct {
	slotToFeeRecipientAndPayloadID map[types.Slot][feeRecipientAndPayloadIDLength]byte
	sync.RWMutex
}

func NewFeeRecipientPayloadIDCache() *FeeRecipientPayloadIDCache {
	return &FeeRecipientPayloadIDCache{
		slotToFeeRecipientAndPayloadID: make(map[types.Slot][feeRecipientAndPayloadIDLength]byte),
	}
}

func (f *FeeRecipientPayloadIDCache) GetFeeRecipientAndPayloadID(slot types.Slot) (feeRecipientAndPayloadID [feeRecipientAndPayloadIDLength]byte, ok bool) {
	f.RLock()
	defer f.RUnlock()
	feeRecipientAndPayloadID, ok = f.slotToFeeRecipientAndPayloadID[slot]
	return
}

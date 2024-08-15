package cache

import (
	"sync"

	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

var (
	mu                                 sync.RWMutex
	cachedSignedExecutionPayloadHeader *enginev1.SignedExecutionPayloadHeader
)

func SaveSignedExecutionPayloadHeader(header *enginev1.SignedExecutionPayloadHeader) {
	mu.Lock()
	defer mu.Unlock()
	cachedSignedExecutionPayloadHeader = header
}

func SignedExecutionPayloadHeader() *enginev1.SignedExecutionPayloadHeader {
	mu.RLock()
	defer mu.RUnlock()
	return eth.CopySignedExecutionPayloadHeader(cachedSignedExecutionPayloadHeader)
}

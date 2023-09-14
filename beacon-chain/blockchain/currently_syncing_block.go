package blockchain

import "sync"

type currentlySyncingBlock struct {
	sync.Mutex
	root [32]byte
}

func (b *currentlySyncingBlock) set(root [32]byte) {
	b.Lock()
	defer b.Unlock()
	b.root = root
}

func (b *currentlySyncingBlock) unset() {
	b.set([32]byte{})
}

func (b *currentlySyncingBlock) get() [32]byte {
	b.Lock()
	defer b.Unlock()
	return b.root
}

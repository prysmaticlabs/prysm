package customtypes

import (
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/math"
	log "github.com/sirupsen/logrus"
)

// TODO:
// - start dividing into chunks only after some threshold?

type Balances struct {
	chunks        [256][]uint64
	len           int
	fullChunkSize uint64
	sharedRefs    map[uint64]*stateutil.Reference
	// TODO: Is a lock needed?
	lock sync.RWMutex
}

func NewBalances(balances []uint64) *Balances {
	chunks, fullChunkSize := buildChunks(balances)
	b := &Balances{
		chunks:        chunks,
		len:           len(balances),
		fullChunkSize: fullChunkSize,
		sharedRefs:    make(map[uint64]*stateutil.Reference),
	}
	return b
}

func (b *Balances) Copy() *Balances {
	b.lock.Lock()
	defer b.lock.Unlock()

	log.Warnf("Copying balances")
	debug.PrintStack()
	var chunks [256][]uint64
	refs := make(map[uint64]*stateutil.Reference)
	// TODO: Can we simply do bCopy.chunks = b.chunks?
	for i, ch := range b.chunks {
		chunks[i] = ch
		r, ok := b.sharedRefs[uint64(i)]
		if ok {
			//log.Warnf("Adding ref to chunk %d, count is now %d", i, r.Refs()+1)
			refs[uint64(i)] = r
			r.AddRef()
		} else {
			//log.Warnf("New ref of 2 for chunk %d", i)
			newRef := stateutil.NewRef(2)
			b.sharedRefs[uint64(i)] = newRef
			refs[uint64(i)] = newRef
		}
	}

	bCopy := &Balances{
		chunks:        chunks,
		len:           b.len,
		fullChunkSize: b.fullChunkSize,
		sharedRefs:    refs,
	}
	return bCopy
}

func (b *Balances) Len() int {
	return b.len
}

func (b *Balances) Value() []uint64 {
	b.lock.RLock()
	defer b.lock.RUnlock()

	// TODO: Log and see how many copies we make
	index := 0
	v := make([]uint64, b.len)
	for _, ch := range b.chunks {
		if len(ch) == 0 {
			break
		}
		copy(v[index:], ch)
		index += len(ch)
	}
	return v
}

func (b *Balances) At(i primitives.ValidatorIndex) (uint64, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	chunkIndex := uint64(i) / b.fullChunkSize
	elemIndex := uint64(i) % b.fullChunkSize
	if chunkIndex >= uint64(len(b.chunks)) || elemIndex >= uint64(len(b.chunks[chunkIndex])) {
		//log.Warnf("chunkIndex: %d, len(chunks): %d, ememIndex: %d, len(chunks[chunkIndex]): %d", chunkIndex, len(b.chunks), elemIndex, len(b.chunks[chunkIndex]))
		return 0, fmt.Errorf("validator index %d is too large", i)
	}
	return b.chunks[chunkIndex][elemIndex], nil
}

func (b *Balances) UpdateAt(i primitives.ValidatorIndex, val uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	//log.Warnf("Updating at index %d", i)
	chunkIndex := uint64(i) / b.fullChunkSize
	elemIndex := uint64(i) % b.fullChunkSize
	if chunkIndex >= uint64(len(b.chunks)) || elemIndex >= uint64(len(b.chunks[chunkIndex])) {
		//log.Warnf("chunkIndex: %d, len(chunks): %d, ememIndex: %d, len(chunks[chunkIndex]): %d", chunkIndex, len(b.chunks), elemIndex, len(b.chunks[chunkIndex]))
		return fmt.Errorf("validator index %d is too large", i)
	}

	ref, ok := b.sharedRefs[chunkIndex]
	if ok {
		if ref.Refs() > 1 {
			//log.Warnf("Copying chunk at index %d, copy size is %d bytes", chunkIndex, len(b.chunks[chunkIndex])*8)
			//debug.PrintStack()
			newChunk := make([]uint64, len(b.chunks[chunkIndex]))
			copy(newChunk, b.chunks[chunkIndex])
			newChunk[elemIndex] = val
			b.chunks[chunkIndex] = newChunk
			//log.Warnf("Removing ref from chunk %d, count is now %d", chunkIndex, ref.Refs()-1)
			ref.MinusRef()
		} else {
			b.chunks[chunkIndex][elemIndex] = val
		}
		delete(b.sharedRefs, chunkIndex)
	} else {
		b.chunks[chunkIndex][elemIndex] = val
	}

	return nil
}

func (b *Balances) Append(val uint64) {
	b.lock.Lock()
	defer b.lock.Unlock()

	chunksAreFull := uint64(len(b.chunks[len(b.chunks)-1])) == b.fullChunkSize
	if chunksAreFull {
		index := uint64(0)
		balances := make([]uint64, b.len+1)
		for i, ch := range b.chunks {
			ref, ok := b.sharedRefs[uint64(i)]
			if ok {
				ref.MinusRef()
				delete(b.sharedRefs, uint64(i))
			}
			copy(balances[index:], ch)
			index += b.fullChunkSize
		}
		balances[len(balances)-1] = val
		b.chunks, b.fullChunkSize = buildChunks(balances)
	} else {
		chunkIndex := uint64(b.len) / b.fullChunkSize
		ref, ok := b.sharedRefs[chunkIndex]
		if ok {
			chunkCopy := make([]uint64, len(b.chunks[chunkIndex]))
			copy(chunkCopy, b.chunks[chunkIndex])
			b.chunks[chunkIndex] = append(chunkCopy, val)
			ref.MinusRef()
			delete(b.sharedRefs, chunkIndex)
		} else {
			b.chunks[chunkIndex] = append(b.chunks[chunkIndex], val)
		}
	}

	b.len++
}

func buildChunks(balances []uint64) ([256][]uint64, uint64) {
	balancesSize := uint64(len(balances))
	fullChunkSize := balancesSize/256 + 1
	var chunks [256][]uint64
	chunkIndex := 0

	for i := uint64(0); i < balancesSize; i += fullChunkSize {
		chunkLen := math.Min(fullChunkSize, balancesSize-i)
		chunks[chunkIndex] = make([]uint64, chunkLen)
		copy(chunks[chunkIndex], balances[i:i+chunkLen])
		chunkIndex++
	}
	for i := chunkIndex; i < 256; i++ {
		chunks[i] = make([]uint64, 0)
	}

	return chunks, fullChunkSize
}

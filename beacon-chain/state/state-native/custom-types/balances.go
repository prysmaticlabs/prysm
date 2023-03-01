package customtypes

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/math"
)

// TODO:
// - start dividing into chunks only after some threshold?

type Balances struct {
	chunks                [256][]uint64
	len                   int
	fullChunkSize         uint64
	sharedChunkReferences map[uint64]*stateutil.Reference
}

func NewBalances(balances []uint64) *Balances {
	chunks, fullChunkSize := buildChunks(balances)
	b := &Balances{
		chunks:                chunks,
		len:                   len(balances),
		fullChunkSize:         fullChunkSize,
		sharedChunkReferences: make(map[uint64]*stateutil.Reference),
	}
	return b
}

func (b *Balances) Copy() *Balances {
	var chunks [256][]uint64
	refs := make(map[uint64]*stateutil.Reference)
	// TODO: Can we simply do bCopy.chunks = b.chunks?
	for i, ch := range b.chunks {
		chunks[i] = ch
		r, ok := b.sharedChunkReferences[uint64(i)]
		if ok {
			refs[uint64(i)] = r
			r.AddRef()
		} else {
			newRef := stateutil.NewRef(2)
			b.sharedChunkReferences[uint64(i)] = newRef
			refs[uint64(i)] = newRef
		}
	}

	bCopy := &Balances{
		chunks:                chunks,
		len:                   b.len,
		fullChunkSize:         b.fullChunkSize,
		sharedChunkReferences: refs,
	}
	return bCopy
}

func (b *Balances) Len() int {
	return b.len
}

func (b *Balances) Value() []uint64 {
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
	chunk := uint64(i) / 256
	indexInChunk := uint64(i) % 256
	if chunk >= uint64(len(b.chunks)) || indexInChunk >= uint64(len(b.chunks[chunk])) {
		return 0, fmt.Errorf("validator index %d is too large", i)
	}
	return b.chunks[chunk][indexInChunk], nil
}

func (b *Balances) UpdateAt(i primitives.ValidatorIndex, val uint64) error {
	chunkIndex := uint64(i) / 256
	elemIndex := uint64(i) % 256
	if chunkIndex >= uint64(len(b.chunks)) || elemIndex >= uint64(len(b.chunks[chunkIndex])) {
		return fmt.Errorf("validator index %d is too large", i)
	}

	ref, ok := b.sharedChunkReferences[chunkIndex]
	if ok {
		if ref.Refs() > 1 {
			newChunk := make([]uint64, len(b.chunks[chunkIndex]))
			copy(newChunk, b.chunks[chunkIndex])
			newChunk[elemIndex] = val
			b.sharedChunkReferences[chunkIndex].MinusRef()
			if ref.Refs() == 1 {
				delete(b.sharedChunkReferences, chunkIndex)
			}
		} else {
			b.chunks[chunkIndex][elemIndex] = val
			delete(b.sharedChunkReferences, chunkIndex)
		}
	} else {
		b.chunks[chunkIndex][elemIndex] = val
	}

	return nil
}

func (b *Balances) Append(val uint64) {
	chunksAreFull := len(b.chunks[len(b.chunks)-1]) == len(b.chunks[0])
	if chunksAreFull {
		index := 0
		balances := make([]uint64, b.len+1)
		for i, ch := range b.chunks {
			ref, ok := b.sharedChunkReferences[uint64(i)]
			if ok {
				ref.MinusRef()
				delete(b.sharedChunkReferences, uint64(i))
			}
			if len(ch) > 0 {
				copy(balances[index:], ch)
				index += len(ch)
			}
		}
		balances[len(balances)-1] = val
		b.chunks, b.fullChunkSize = buildChunks(balances)
	} else {
		fullChunkLen := len(b.chunks[0])
		chunkIndex := b.len / fullChunkLen
		chunkCopy := make([]uint64, len(b.chunks[chunkIndex]))
		copy(chunkCopy, b.chunks[chunkIndex])
		b.chunks[chunkIndex] = append(chunkCopy, val)
		ref, ok := b.sharedChunkReferences[uint64(chunkIndex)]
		if ok {
			ref.MinusRef()
			delete(b.sharedChunkReferences, uint64(chunkIndex))
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
		chunks[chunkIndex] = make([]uint64, 0)
	}

	return chunks, fullChunkSize
}

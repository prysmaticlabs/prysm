package sync

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
)

type blockAndBlobs struct {
	blk   interfaces.ReadOnlySignedBeaconBlock
	blobs []*eth.BlobSidecar
}

type blockAndBlocksQueue struct {
	lock  sync.RWMutex
	queue map[[32]byte]blockAndBlobs
}

func newBlockAndBlobs() *blockAndBlocksQueue {
	return &blockAndBlocksQueue{
		queue: make(map[[32]byte]blockAndBlobs),
	}
}

func (q *blockAndBlocksQueue) addBlock(b interfaces.ReadOnlySignedBeaconBlock) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	if b.Version() < version.Deneb {
		return errors.New("block version is not supported")
	}

	r, err := b.Block().HashTreeRoot()
	if err != nil {
		return err
	}
	bnb, ok := q.queue[r]
	if !ok {
		q.queue[r] = blockAndBlobs{
			blk:   b,
			blobs: make([]*eth.BlobSidecar, 0, 4),
		}
		return nil
	}
	bnb.blk = b
	q.queue[r] = bnb

	return nil
}

func (q *blockAndBlocksQueue) addBlob(b *eth.BlobSidecar) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	r := bytesutil.ToBytes32(b.BlockRoot)

	bnb, ok := q.queue[r]
	if !ok {
		q.queue[r] = blockAndBlobs{
			blobs: make([]*eth.BlobSidecar, 0, 4),
		}
		bnb = q.queue[r]
	}
	bnb.blobs = append(bnb.blobs, b)
	q.queue[r] = bnb

	return nil
}

func (q *blockAndBlocksQueue) getBlock(r [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
	q.lock.RLock()
	defer q.lock.RUnlock()

	bnb, ok := q.queue[r]
	if !ok {
		return nil, errors.New("block does not exist")
	}
	if bnb.blk == nil {
		return nil, errors.New("block does not exist")
	}
	return bnb.blk, nil
}

func (q *blockAndBlocksQueue) getBlob(r [32]byte, i uint64) (*eth.BlobSidecar, error) {
	q.lock.RLock()
	defer q.lock.RUnlock()

	if i >= params.MaxBlobsPerBlock {
		return nil, errors.New("request out of bounds")
	}

	bnb, ok := q.queue[r]
	if !ok {
		return nil, errors.New("blob does not exist")
	}
	for _, blob := range bnb.blobs {
		if i == blob.Index {
			return blob, nil
		}
	}
	return nil, errors.New("blob does not exist")
}

func (q *blockAndBlocksQueue) delete(r [32]byte) {
	q.lock.Lock()
	defer q.lock.Unlock()

	delete(q.queue, r)
}

func (q *blockAndBlocksQueue) canImport(r [32]byte) (bool, error) {
	q.lock.RLock()
	defer q.lock.RUnlock()

	bnb, ok := q.queue[r]
	if !ok {
		return false, nil
	}

	if bnb.blk == nil || bnb.blk.IsNil() {
		return false, nil
	}

	commitments, err := bnb.blk.Block().Body().BlobKzgCommitments()
	if err != nil {
		return false, err
	}

	// TODO: enhance check to ensure that block commitments match blob

	return len(commitments) == len(bnb.blobs), nil
}

func (q *blockAndBlocksQueue) missingRootAndIndex(ctx context.Context) (map[[32]byte][]uint64, error) {
	q.lock.RLock()
	defer q.lock.RUnlock()

	m := make(map[[32]byte][]uint64)

	for r, b := range q.queue {
		kzgs, err := b.blk.Block().Body().BlobKzgCommitments()
		if err != nil {
			return nil, err
		}
		has := make(map[uint64]bool)
		for _, blob := range b.blobs {
			has[blob.Index] = true
		}
		for i := range kzgs {
			if has[uint64(i)] {
				continue
			}
			m[r] = append(m[r], uint64(i))
		}
	}
	return m, nil
}

package sync

import (
	"context"
	"sort"
	"time"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var processPendingBlocksPeriod = time.Duration(params.BeaconConfig().SecondsPerSlot/3) * time.Second

// processes pending blocks queue on every processPendingBlocksPeriod
func (r *RegularSync) processPendingBlocksQueue() {
	ticker := time.NewTicker(processPendingBlocksPeriod)
	for {
		ctx := context.TODO()
		select {
		case <-ticker.C:
			r.processPendingBlocks(ctx)
		case <-r.ctx.Done():
			log.Debug("p2p context is closed, exiting routine")
			break

		}
	}
}

// processes the block tree inside the queue
func (r *RegularSync) processPendingBlocks(ctx context.Context) error {
	pids := r.peerIDs()
	slots := r.sortedPendingSlots()

	for _, s := range slots {
		r.slotToPendingBlocksLock.RLock()
		b := r.slotToPendingBlocks[uint64(s)]
		r.slotToPendingBlocksLock.RUnlock()

		r.seenPendingBlocksLock.RLock()
		inPendingQueue := r.seenPendingBlocks[bytesutil.ToBytes32(b.ParentRoot)]
		r.seenPendingBlocksLock.RUnlock()

		inDB := r.db.HasBlock(ctx, bytesutil.ToBytes32(b.ParentRoot))
		hasPeer := len(pids) != 0

		// Only request for missing parent block if it's not in DB, not in pending cache
		// and has peer in the peer list.
		if !inPendingQueue && !inDB && hasPeer {
			log.Infof("Request parent of block %d", b.Slot)
			req := [][32]byte{bytesutil.ToBytes32(b.ParentRoot)}
			if err := r.sendRecentBeaconBlocksRequest(ctx, req, pids[0]); err != nil {
				log.Errorf("Could not send recent block request: %v", err)
			}
			continue
		}

		if !inDB {
			continue
		}

		if err := r.chain.ReceiveBlockNoPubsub(ctx, b); err != nil {
			log.Errorf("Could not process block from slot %d: %v", b.Slot, err)
		}

		r.slotToPendingBlocksLock.Lock()
		r.seenPendingBlocksLock.Lock()
		delete(r.slotToPendingBlocks, uint64(s))
		delete(r.seenPendingBlocks, bytesutil.ToBytes32(b.ParentRoot))
		r.slotToPendingBlocksLock.Unlock()
		r.seenPendingBlocksLock.Unlock()

		log.Infof("Processed ancestor block %d and cleared pending block cache", s)
	}
	return nil
}

func (r *RegularSync) peerIDs() []peer.ID {
	hellos := r.Hellos()
	pids := make([]peer.ID, 0, len(hellos))
	for pid := range hellos {
		pids = append(pids, pid)
	}
	return pids
}

func (r *RegularSync) sortedPendingSlots() []int {
	slots := make([]int, 0, len(r.slotToPendingBlocks))
	for s := range r.slotToPendingBlocks {
		slots = append(slots, int(s))
	}
	sort.Ints(slots)

	return slots
}

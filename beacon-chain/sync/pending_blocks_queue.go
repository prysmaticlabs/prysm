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
	// Construct a sorted list of slots from outstanding pending blocks
	hellos := r.Hellos()
	pids := make([]peer.ID, 0, len(hellos))
	for pid := range hellos {
		pids = append(pids, pid)
	}

	slots := make([]int, 0, len(r.slotToPendingBlocks))
	for s := range r.slotToPendingBlocks {
		slots = append(slots, int(s))
	}
	sort.Ints(slots)

	for _, s := range slots {
		b := r.slotToPendingBlocks[uint64(s)]
		inPendingQueue := r.seenPendingBlocks[bytesutil.ToBytes32(b.ParentRoot)]
		inDB := r.db.HasBlock(ctx, bytesutil.ToBytes32(b.ParentRoot))
		hasPeer := len(pids) != 0

		if !inPendingQueue && !inDB && hasPeer {
			log.Infof("Request parent of block %d", b.Slot)
			req := [][32]byte{bytesutil.ToBytes32(b.ParentRoot)}
			// Always request from the first peer, should be upgraded with round robin
			if err := r.sendRecentBeaconBlocksRequest(ctx, req, pids[0]); err != nil {
				log.Errorf("Could not send recent block request: %v", err)
				continue
			}
			continue
		}

		if !inDB {
			continue
		}

		if err := r.chain.ReceiveBlockNoPubsub(ctx, b); err != nil {
			log.Errorf("Could not process block from slot %d: %v", b.Slot, err)
		}

		delete(r.slotToPendingBlocks, uint64(s))
		delete(r.seenPendingBlocks, bytesutil.ToBytes32(b.ParentRoot))

		log.Infof("Processed ancestor block %d and cleared pending block cache", s)
	}
	return nil
}

package initialsync

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// nonSkippedSlotAfter checks slots after the given one in an attempt to find a non-empty future slot.
// For efficiency only one random slot is checked per epoch, so returned slot might not be the first
// non-skipped slot. This shouldn't be a problem, as in case of adversary peer, we might get incorrect
// data anyway, so code that relies on this function must be robust enough to re-request, if no progress
// is possible with a returned value.
func (f *blocksFetcher) nonSkippedSlotAfter(ctx context.Context, slot uint64) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "initialsync.nonSkippedSlotAfter")
	defer span.End()

	headEpoch := helpers.SlotToEpoch(f.headFetcher.HeadSlot())
	finalizedEpoch, peers := f.p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, headEpoch)
	log.WithFields(logrus.Fields{
		"start":          slot,
		"headEpoch":      headEpoch,
		"finalizedEpoch": finalizedEpoch,
	}).Debug("Searching for non-skipped slot")
	// Exit early, if no peers with high enough finalized epoch are found.
	if finalizedEpoch <= headEpoch {
		return 0, errSlotIsTooHigh
	}
	var err error
	if featureconfig.Get().EnablePeerScorer {
		peers, err = f.filterScoredPeers(ctx, peers, peersPercentagePerRequest)
	} else {
		peers, err = f.filterPeers(peers, peersPercentagePerRequest)
	}
	if err != nil {
		return 0, err
	}
	if len(peers) == 0 {
		return 0, errNoPeersAvailable
	}

	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	pidInd := 0

	fetch := func(pid peer.ID, start, count, step uint64) (uint64, error) {
		req := &p2ppb.BeaconBlocksByRangeRequest{
			StartSlot: start,
			Count:     count,
			Step:      step,
		}
		blocks, err := f.requestBlocks(ctx, req, pid)
		if err != nil {
			return 0, err
		}
		if len(blocks) > 0 {
			for _, block := range blocks {
				if block.Block.Slot > slot {
					return block.Block.Slot, nil
				}
			}
		}
		return 0, nil
	}

	// Start by checking several epochs fully, w/o resorting to random sampling.
	start := slot + 1
	end := start + nonSkippedSlotsFullSearchEpochs*slotsPerEpoch
	for ind := start; ind < end; ind += slotsPerEpoch {
		nextSlot, err := fetch(peers[pidInd%len(peers)], ind, slotsPerEpoch, 1)
		if err != nil {
			return 0, err
		}
		if nextSlot > slot {
			return nextSlot, nil
		}
		pidInd++
	}

	// Quickly find the close enough epoch where a non-empty slot definitely exists.
	// Only single random slot per epoch is checked - allowing to move forward relatively quickly.
	slot = slot + nonSkippedSlotsFullSearchEpochs*slotsPerEpoch
	upperBoundSlot := helpers.StartSlot(finalizedEpoch + 1)
	for ind := slot + 1; ind < upperBoundSlot; ind += (slotsPerEpoch * slotsPerEpoch) / 2 {
		start := ind + uint64(f.rand.Intn(int(slotsPerEpoch)))
		nextSlot, err := fetch(peers[pidInd%len(peers)], start, slotsPerEpoch/2, slotsPerEpoch)
		if err != nil {
			return 0, err
		}
		pidInd++
		if nextSlot > slot && upperBoundSlot >= nextSlot {
			upperBoundSlot = nextSlot
			break
		}
	}

	// Epoch with non-empty slot is located. Check all slots within two nearby epochs.
	if upperBoundSlot > slotsPerEpoch {
		upperBoundSlot -= slotsPerEpoch
	}
	upperBoundSlot = helpers.StartSlot(helpers.SlotToEpoch(upperBoundSlot))
	nextSlot, err := fetch(peers[pidInd%len(peers)], upperBoundSlot, slotsPerEpoch*2, 1)
	if err != nil {
		return 0, err
	}
	if nextSlot < slot || helpers.StartSlot(finalizedEpoch+1) < nextSlot {
		return 0, errors.New("invalid range for non-skipped slot")
	}
	return nextSlot, nil
}

// bestFinalizedSlot returns the highest finalized slot of the majority of connected peers.
func (f *blocksFetcher) bestFinalizedSlot() uint64 {
	headEpoch := helpers.SlotToEpoch(f.headFetcher.HeadSlot())
	finalizedEpoch, _ := f.p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, headEpoch)
	return helpers.StartSlot(finalizedEpoch)
}

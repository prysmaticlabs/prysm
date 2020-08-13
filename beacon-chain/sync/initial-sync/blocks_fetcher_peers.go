package initialsync

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	scorers "github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// getPeerLock returns peer lock for a given peer. If lock is not found, it is created.
func (f *blocksFetcher) getPeerLock(pid peer.ID) *peerLock {
	f.Lock()
	defer f.Unlock()
	if lock, ok := f.peerLocks[pid]; ok {
		lock.accessed = roughtime.Now()
		return lock
	}
	f.peerLocks[pid] = &peerLock{
		Mutex:    sync.Mutex{},
		accessed: roughtime.Now(),
	}
	return f.peerLocks[pid]
}

// removeStalePeerLocks is a cleanup procedure which removes stale locks.
func (f *blocksFetcher) removeStalePeerLocks(age time.Duration) {
	f.Lock()
	defer f.Unlock()
	for peerID, lock := range f.peerLocks {
		if time.Since(lock.accessed) >= age {
			lock.Lock()
			delete(f.peerLocks, peerID)
			lock.Unlock()
		}
	}
}

// selectFailOverPeer randomly selects fail over peer from the list of available peers.
func (f *blocksFetcher) selectFailOverPeer(excludedPID peer.ID, peers []peer.ID) (peer.ID, error) {
	if len(peers) == 0 {
		return "", errNoPeersAvailable
	}
	if len(peers) == 1 && peers[0] == excludedPID {
		return "", errNoPeersAvailable
	}

	ind := f.rand.Int() % len(peers)
	if peers[ind] == excludedPID {
		return f.selectFailOverPeer(excludedPID, append(peers[:ind], peers[ind+1:]...))
	}
	return peers[ind], nil
}

// waitForMinimumPeers spins and waits up until enough peers are available.
func (f *blocksFetcher) waitForMinimumPeers(ctx context.Context) ([]peer.ID, error) {
	required := params.BeaconConfig().MaxPeersToSync
	if flags.Get().MinimumSyncPeers < required {
		required = flags.Get().MinimumSyncPeers
	}
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		headEpoch := helpers.SlotToEpoch(f.headFetcher.HeadSlot())
		_, peers := f.p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, headEpoch)
		if len(peers) >= required {
			return peers, nil
		}
		log.WithFields(logrus.Fields{
			"suitable": len(peers),
			"required": required}).Info("Waiting for enough suitable peers before syncing")
		time.Sleep(handshakePollingInterval)
	}
}

// filterPeers returns transformed list of peers,
// weight ordered or randomized, constrained if necessary (only percentage of peers returned).
func (f *blocksFetcher) filterPeers(peers []peer.ID, peersPercentage float64) ([]peer.ID, error) {
	if len(peers) == 0 {
		return peers, nil
	}

	// Shuffle peers to prevent a bad peer from
	// stalling sync with invalid blocks.
	f.rand.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

	// Select sub-sample from peers (honoring min-max invariants).
	peers = trimPeers(peers, peersPercentage)

	// Order peers by remaining capacity, effectively turning in-order
	// round robin peer processing into a weighted one (peers with higher
	// remaining capacity are preferred). Peers with the same capacity
	// are selected at random, since we have already shuffled peers
	// at this point.
	sort.SliceStable(peers, func(i, j int) bool {
		cap1 := f.rateLimiter.Remaining(peers[i].String())
		cap2 := f.rateLimiter.Remaining(peers[j].String())
		return cap1 > cap2
	})

	return peers, nil
}

// filterScoredPeers returns transformed list of peers,
// weight sorted by scores and capacity remaining. List can be constrained using peersPercentage,
// where only percentage of peers are returned.
func (f *blocksFetcher) filterScoredPeers(ctx context.Context, peers []peer.ID, peersPercentage float64) ([]peer.ID, error) {
	ctx, span := trace.StartSpan(ctx, "initialsync.filterScoredPeers")
	defer span.End()

	if len(peers) == 0 {
		return peers, nil
	}

	// Sort peers using both block provider score and, custom, capacity based score (see
	// peerFilterCapacityWeight if you want to give different weights to provider's and capacity
	// scores).
	// Scores produced are used as weights, so peers are ordered probabilistically i.e. peer with
	// a higher score has higher chance to end up higher in the list.
	scorer := f.p2p.Peers().Scorers().BlockProviderScorer()
	peers = scorer.WeightSorted(f.rand, peers, func(peerID peer.ID, blockProviderScore float64) float64 {
		remaining, capacity := float64(f.rateLimiter.Remaining(peerID.String())), float64(f.rateLimiter.Capacity())
		// When no capacity for a good peer left, allow less performant peer to take a chance.
		if remaining < float64(f.blocksPerSecond) {
			return 0.0
		}
		capScore := remaining / capacity
		overallScore := blockProviderScore*(1.0-f.capacityWeight) + capScore*f.capacityWeight
		return math.Round(overallScore*scorers.ScoreRoundingFactor) / scorers.ScoreRoundingFactor
	})
	peers = trimPeers(peers, peersPercentage)

	return peers, nil
}

// trimPeers limits peer list, returning only specified percentage of peers.
// Takes system constraints into account (min/max peers to sync).
func trimPeers(peers []peer.ID, peersPercentage float64) []peer.ID {
	required := params.BeaconConfig().MaxPeersToSync
	if flags.Get().MinimumSyncPeers < required {
		required = flags.Get().MinimumSyncPeers
	}
	// Weak/slow peers will be pushed down the list and trimmed since only percentage of peers is selected.
	limit := uint64(math.Round(float64(len(peers)) * peersPercentage))
	// Limit cannot be less that minimum peers required by sync mechanism.
	limit = mathutil.Max(limit, uint64(required))
	// Limit cannot be higher than number of peers available (safe-guard).
	limit = mathutil.Min(limit, uint64(len(peers)))
	return peers[:limit]
}

package initialsync

import (
	"context"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/peerstatus"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

const blockBatchSize = 64

// Round Robin sync looks at the latest peer statuses and syncs with the highest
// finalized peer,
//
// Step 1 - Sync to finalized epoch.
// Sync with peers of lowest finalized root with epoch greater than head state.
//
// Step 2 - Sync to head from finalized epoch.
// Using the finalized root as the head_block_root and the epoch start slot
// after the finalized epoch, request blocks to head from some subset of peers
// where step = 1.
func (s *InitialSync) roundRobinSync(genesis time.Time) error {
	ctx := context.Background()

	// Sync to finalized epoch.
	for s.chain.HeadSlot() < helpers.StartSlot(highestFinalizedEpoch()+1) {
		root, _, peers := highestFinalized()

		req := &p2ppb.BeaconBlocksByRangeRequest{
			HeadBlockRoot: root,
			StartSlot:     s.chain.HeadSlot() + 1,
			Count:         blockBatchSize,
			Step:          uint64(len(peers)),
		}

		var blocks []*eth.BeaconBlock
		for _, pid := range peers {
			log.WithField("req", req).WithField("peer", pid.Pretty()).Debug(
				"Sending batch block request",
			)
			stream, err := s.p2p.Send(ctx, req, pid)
			if err != nil {
				// TODO: Retry request in round robin with other peers, if possible.
				return errors.Wrap(err, "failed to send request to peer")
			}

			// TODO: Abstract inner logic
			// TODO: Requests in parallel.
			// TODO: Stream deadlines.

			code, errMsg, err := sync.ReadStatusCode(stream, s.p2p.Encoding())
			if err != nil {
				return errors.Wrap(err, "failed to read response status")
			}
			if code != 0 {
				return errors.New(errMsg)
			}

			resp := make([]*eth.BeaconBlock, 0)
			if err := s.p2p.Encoding().DecodeWithLength(stream, &resp); err != nil {
				return errors.Wrap(err, "failed to decode response")
			}

			blocks = append(blocks, resp...)

			req.StartSlot++
		}

		sort.Slice(blocks, func(i, j int) bool {
			return blocks[i].Slot < blocks[j].Slot
		})

		for _, blk := range blocks {
			if err := s.chain.ReceiveBlockNoPubsubForkchoice(ctx, blk); err != nil {
				return err
			}
		}
	}

	log.Debug("Synced to finalized epoch. Syncing blocks to head slot now.")

	// TODO: Step 2 - sync to head.
	best := bestPeer()
	root, _, _ := highestFinalized()
	req := &p2ppb.BeaconBlocksByRangeRequest{
		HeadBlockRoot:        root,
		StartSlot:            s.chain.HeadSlot() + 1,
		Count:                slotsSinceGenesis(genesis) - s.chain.HeadSlot() + 1,
		Step:                 1,
	}

	log.WithField("req", req).WithField("peer", best.Pretty()).Debug(
		"Sending batch block request",
	)

	stream, err := s.p2p.Send(ctx, req, best)
	if err != nil {
		return errors.Wrap(err, "failed to send request to peer")
	}

	code, errMsg, err := sync.ReadStatusCode(stream, s.p2p.Encoding())
	if err != nil {
		return errors.Wrap(err, "failed to read status code")
	}
	if code != 0 {
		return errors.New(errMsg)
	}

	resp := make([]*eth.BeaconBlock, 0)
	if err := s.p2p.Encoding().DecodeWithLength(stream, &resp); err != nil {
		return errors.Wrap(err, "failed to decode response")
	}

	for _, blk := range resp {
		if err := s.chain.ReceiveBlockNoPubsubForkchoice(ctx, blk); err != nil {
			return err
		}
	}

	return nil
}

// highestFinalizedEpoch as reported by peers.
func highestFinalizedEpoch() uint64 {
	var finalizedEpoch uint64
	for _, k := range peerstatus.Keys() {
		s := peerstatus.Get(k)
		finalizedEpoch = mathutil.Max(s.FinalizedEpoch, finalizedEpoch)
	}

	return finalizedEpoch
}

func highestFinalized() ([]byte, uint64, []peer.ID) {
	finalized := make(map[[32]byte]uint64)
	rootToEpoch := make(map[[32]byte]uint64)
	for _, k := range peerstatus.Keys() {
		s := peerstatus.Get(k)
		r := bytesutil.ToBytes32(s.FinalizedRoot)
		finalized[r]++
		rootToEpoch[r] = s.FinalizedEpoch
	}

	var mostVotedFinalizedRoot [32]byte
	var mostVotes uint64
	for root, count := range finalized {
		if count > mostVotes {
			mostVotes = count
			mostVotedFinalizedRoot = root
		}
	}

	var pids []peer.ID
	for _, k := range peerstatus.Keys() {
		s := peerstatus.Get(k)
		if s.FinalizedEpoch >= rootToEpoch[mostVotedFinalizedRoot] {
			pids = append(pids, k)
		}
	}

	return mostVotedFinalizedRoot[:], rootToEpoch[mostVotedFinalizedRoot], pids
}

func bestPeer() peer.ID {
	var best peer.ID
	var bestSlot uint64
	for _, k := range peerstatus.Keys() {
		s := peerstatus.Get(k)
		if s.HeadSlot >= bestSlot {
			bestSlot = s.HeadSlot
			best = k
		}
	}
	return best
}
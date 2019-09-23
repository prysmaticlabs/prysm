package initialsync

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	prysmsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/peerstatus"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

const blockBatchSize = 64
const readTimeout = 10 * time.Second

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

	var requestBlocks = func(req *p2ppb.BeaconBlocksByRangeRequest, pid peer.ID) ([]*eth.BeaconBlock, error) {
		log.WithField("peer", pid.Pretty()).WithField("req", req).Debug("requesting blocks")
		stream, err := s.p2p.Send(ctx, req, pid)
		if err != nil {
			return nil, errors.Wrap(err, "failed to send request to peer")
		}

		if err := stream.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			return nil, err
		}

		code, errMsg, err := prysmsync.ReadStatusCode(stream, s.p2p.Encoding())
		if err != nil {
			return nil, errors.Wrap(err, "failed to read response status")
		}
		if code != 0 {
			return nil, errors.New(errMsg)
		}

		resp := make([]*eth.BeaconBlock, 0)
		if err := s.p2p.Encoding().DecodeWithLength(stream, &resp); err != nil {
			return nil, errors.Wrap(err, "failed to decode response")
		}
		return resp, nil
	}

	// Step 1 - Sync to end of finalized epoch.
	for s.chain.HeadSlot() < helpers.StartSlot(highestFinalizedEpoch()+1) {
		root, finalizedEpoch, peers := highestFinalized()

		var blocks []*eth.BeaconBlock
		var request func(start uint64, step uint64, count uint64, peers []peer.ID) ([]*eth.BeaconBlock, error)
		request = func(start uint64, step uint64, count uint64, peers []peer.ID) ([]*eth.BeaconBlock, error) {
			if len(peers) == 0 {
				return nil, errors.WithStack(errors.New("no peers left to request blocks"))
			}

			for i, pid := range peers {
				start := start + uint64(i)*step
				step := step * uint64(len(peers))
				count := uint64(math.Min(float64(count), float64((helpers.StartSlot(finalizedEpoch+1)-start)/step)))
				if count == 0 {
					count = 1
				}
				// If the count was divided by an odd number of peers, there will be one block
				// missing from the first request so we accommodate that scenario.
				if i == 0 && len(peers)%2 == 1 && count%2 == 1 {
					count++
				}
				req := &p2ppb.BeaconBlocksByRangeRequest{
					HeadBlockRoot: root,
					StartSlot:     start,
					Count:         count,
					Step:          step,
				}

				resp, err := requestBlocks(req, pid)
				log.WithField("peer", pid.Pretty()).Debugf("Received %d blocks", len(resp))
				if err != nil {
					// try other peers
					ps := append(peers[:i], peers[i+1:]...)
					log.WithError(err).WithField("remaining peers", len(ps)).WithField("peer", pid.Pretty()).Debug("Request failed, trying to round robin with other peers")
					if len(ps) == 0 {
						return nil, errors.WithStack(errors.New("no peers left to request blocks"))
					}
					_, err = request(start, uint64(len(peers)), count/uint64(len(ps)), ps)
					if err != nil {
						return nil, err
					}
				}
				blocks = append(blocks, resp...)
			}

			return blocks, nil
		}

		blocks, err := request(
			s.chain.HeadSlot()+1, // start
			1,                    // step
			blockBatchSize,       // count
			peers,
		)
		if err != nil {
			return err
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

	if s.chain.HeadSlot() == slotsSinceGenesis(genesis) {
		return nil
	}

	// Step 2 - sync to head.
	best := bestPeer()
	root, _, _ := highestFinalized()
	req := &p2ppb.BeaconBlocksByRangeRequest{
		HeadBlockRoot: root,
		StartSlot:     s.chain.HeadSlot() + 1,
		Count:         slotsSinceGenesis(genesis) - s.chain.HeadSlot() + 1,
		Step:          1,
	}

	log.WithField("req", req).WithField("peer", best.Pretty()).Debug(
		"Sending batch block request",
	)

	resp, err := requestBlocks(req, best)
	if err != nil {
		return err
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

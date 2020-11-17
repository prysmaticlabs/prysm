package scorers_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers/scorers"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestScorers_PeerStatus_Score(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//batchSize := uint64(flags.Get().BlockBatchLimit)
	tests := []struct {
		name   string
		update func(scorer *scorers.PeerStatusScorer)
		check  func(scorer *scorers.PeerStatusScorer)
	}{
		{
			name: "nonexistent peer",
			update: func(scorer *scorers.PeerStatusScorer) {
				scorer.SetHeadSlot(64)
			},
			check: func(scorer *scorers.PeerStatusScorer) {
				assert.Equal(t, 0.0, scorer.Score("peer1"), "Unexpected score")
			},
		},
		{
			name: "existent bad peer",
			update: func(scorer *scorers.PeerStatusScorer) {
				scorer.SetHeadSlot(0)
				scorer.SetPeerStatus("peer1", &pb.Status{
					HeadRoot: make([]byte, 32),
					HeadSlot: 64,
				}, p2ptypes.ErrWrongForkDigestVersion)
			},
			check: func(scorer *scorers.PeerStatusScorer) {
				assert.Equal(t, scorers.BadPeerScore, scorer.Score("peer1"), "Unexpected score")
			},
		},
		{
			name: "existent peer no head slot for the host node is known",
			update: func(scorer *scorers.PeerStatusScorer) {
				scorer.SetHeadSlot(0)
				scorer.SetPeerStatus("peer1", &pb.Status{
					HeadRoot: make([]byte, 32),
					HeadSlot: 64,
				}, nil)
			},
			check: func(scorer *scorers.PeerStatusScorer) {
				assert.Equal(t, 1.0, scorer.Score("peer1"), "Unexpected score")
			},
		},
		{
			name: "existent peer head is before ours",
			update: func(scorer *scorers.PeerStatusScorer) {
				scorer.SetHeadSlot(128)
				scorer.SetPeerStatus("peer1", &pb.Status{
					HeadRoot: make([]byte, 32),
					HeadSlot: 64,
				}, nil)
			},
			check: func(scorer *scorers.PeerStatusScorer) {
				assert.Equal(t, 0.0, scorer.Score("peer1"), "Unexpected score")
			},
		},
		{
			name: "existent peer partial score",
			update: func(scorer *scorers.PeerStatusScorer) {
				headSlot := uint64(128)
				scorer.SetHeadSlot(headSlot)
				scorer.SetPeerStatus("peer1", &pb.Status{
					HeadRoot: make([]byte, 32),
					HeadSlot: headSlot + 64,
				}, nil)
				// Set another peer to a higher score.
				scorer.SetPeerStatus("peer2", &pb.Status{
					HeadRoot: make([]byte, 32),
					HeadSlot: headSlot + 128,
				}, nil)
			},
			check: func(scorer *scorers.PeerStatusScorer) {
				headSlot := uint64(128)
				assert.Equal(t, float64(headSlot+64)/float64(headSlot+128), scorer.Score("peer1"), "Unexpected score")
			},
		},
		{
			name: "existent peer full score",
			update: func(scorer *scorers.PeerStatusScorer) {
				headSlot := uint64(128)
				scorer.SetHeadSlot(headSlot)
				scorer.SetPeerStatus("peer1", &pb.Status{
					HeadRoot: make([]byte, 32),
					HeadSlot: headSlot + 64,
				}, nil)
			},
			check: func(scorer *scorers.PeerStatusScorer) {
				assert.Equal(t, 1.0, scorer.Score("peer1"), "Unexpected score")
			},
		},
		{
			name: "existent peer no max known slot",
			update: func(scorer *scorers.PeerStatusScorer) {
				scorer.SetHeadSlot(0)
				scorer.SetPeerStatus("peer1", &pb.Status{
					HeadRoot: make([]byte, 32),
					HeadSlot: 0,
				}, nil)
			},
			check: func(scorer *scorers.PeerStatusScorer) {
				assert.Equal(t, 0.0, scorer.Score("peer1"), "Unexpected score")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
				PeerLimit:    30,
				ScorerParams: &scorers.Config{},
			})
			scorer := peerStatuses.Scorers().PeerStatusScorer()
			if tt.update != nil {
				tt.update(scorer)
			}
			tt.check(scorer)
		})
	}
}

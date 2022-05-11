//go:build go1.18
// +build go1.18

package sync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	gcache "github.com/patrickmn/go-cache"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

func FuzzValidateBeaconBlockPubSub(f *testing.F) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Service{
		cfg: &config{
			initialSync: &mockSync.Sync{IsSyncing: false},
			chain:       &chainMock.ChainService{},
		},
		ctx:                  ctx,
		cancel:               cancel,
		slotToPendingBlocks:  gcache.New(time.Second, 2*time.Second),
		seenPendingBlocks:    make(map[[32]byte]bool),
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
	}
	validTopic := fmt.Sprintf(p2p.BlockSubnetTopicFormat, []byte{0xb5, 0x30, 0x3f, 0x2a}) + "/" + encoder.ProtocolSuffixSSZSnappy
	f.Add("junk", []byte("junk"), []byte("junk"), []byte("junk"), []byte(validTopic), []byte("junk"), []byte("junk"))
	f.Fuzz(func(t *testing.T, pid string, from, data, seqno, topic, signature, key []byte) {
		r.cfg.p2p = p2ptest.NewFuzzTestP2P()
		r.rateLimiter = newRateLimiter(r.cfg.p2p)
		strTop := string(topic)
		msg := &pubsub.Message{
			Message: &pb.Message{
				From:      from,
				Data:      data,
				Seqno:     seqno,
				Topic:     &strTop,
				Signature: signature,
				Key:       key,
			},
		}
		_, err := r.validateBeaconBlockPubSub(ctx, peer.ID(pid), msg)
		_ = err
	})
}

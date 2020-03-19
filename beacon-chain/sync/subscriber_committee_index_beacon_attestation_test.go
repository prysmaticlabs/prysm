package sync

import (
	"context"
	"testing"
	"time"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestService_committeeIndexBeaconAttestationSubscriber_ValidMessage(t *testing.T) {
	p := p2ptest.NewTestP2P(t)

	ctx := context.Background()
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	s, sKeys := testutil.DeterministicGenesisState(t, 64 /*validators*/)
	if err := s.SetGenesisTime(uint64(time.Now().Unix())); err != nil {
		t.Fatal(err)
	}
	blk, err := testutil.GenerateFullBlock(s, sKeys, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	root, err := ssz.HashTreeRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}
	savedState, _ := beaconstate.InitializeFromProto(&pb.BeaconState{})
	db.SaveState(context.Background(), savedState, root)

	r := &Service{
		attPool: attestations.NewPool(),
		chain: &mock.ChainService{
			State:            s,
			Genesis:          time.Now(),
			ValidAttestation: true,
		},
		chainStarted:        true,
		p2p:                 p,
		db:                  db,
		ctx:                 ctx,
		stateNotifier:       (&mock.ChainService{}).StateNotifier(),
		attestationNotifier: (&mock.ChainService{}).OperationNotifier(),
		initialSync:         &mockSync.Sync{IsSyncing: false},
	}
	r.registerSubscribers()
	r.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Initialized,
		Data: &statefeed.InitializedData{
			StartTime: time.Now(),
		},
	})

	att := &eth.Attestation{
		Data: &eth.AttestationData{
			Slot:            0,
			BeaconBlockRoot: root[:],
		},
		AggregationBits: bitfield.Bitlist{0b0101},
		Signature:       sKeys[0].Sign([]byte("foo"), 0).Marshal(),
	}

	p.ReceivePubSub("/eth2/committee_index0_beacon_attestation", att)

	time.Sleep(time.Second)

	ua := r.attPool.UnaggregatedAttestations()
	if len(ua) == 0 {
		t.Error("No attestations put into pool")
	}
}

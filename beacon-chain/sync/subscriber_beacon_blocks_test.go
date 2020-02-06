package sync

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestRegularSyncBeaconBlockSubscriber_FilterByFinalizedEpoch(t *testing.T) {
	hook := logTest.NewGlobal()
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)

	s, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		FinalizedCheckpoint: &ethpb.Checkpoint{Epoch: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	parent := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(context.Background(), parent); err != nil {
		t.Fatal(err)
	}
	parentRoot, _ := ssz.HashTreeRoot(parent.Block)
	chain := &mock.ChainService{State: s}
	r := &Service{
		db:            db,
		chain:         chain,
		blockNotifier: chain.BlockNotifier(),
		attPool:       attestations.NewPool(),
	}

	b := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: parentRoot[:], Body: &ethpb.BeaconBlockBody{}},
	}
	if err := r.beaconBlockSubscriber(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Received a block older than finalized checkpoint, 1 < %d", params.BeaconConfig().SlotsPerEpoch))

	hook.Reset()
	b.Block.Slot = params.BeaconConfig().SlotsPerEpoch
	if err := r.beaconBlockSubscriber(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	testutil.AssertLogsDoNotContain(t, hook, "Received a block older than finalized checkpoint")
}

func TestDeleteAttsInPool(t *testing.T) {
	r := &Service{
		attPool: attestations.NewPool(),
	}
	att1 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1101}}
	att2 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1110}}
	att3 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1011}}
	att4 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1001}}
	if err := r.attPool.SaveAggregatedAttestation(att1); err != nil {
		t.Fatal(err)
	}
	if err := r.attPool.SaveAggregatedAttestation(att2); err != nil {
		t.Fatal(err)
	}
	if err := r.attPool.SaveAggregatedAttestation(att3); err != nil {
		t.Fatal(err)
	}
	if err := r.attPool.SaveUnaggregatedAttestation(att4); err != nil {
		t.Fatal(err)
	}

	// Seen 1, 3 and 4 in block.
	if err := r.deleteAttsInPool([]*ethpb.Attestation{att1, att3, att4}); err != nil {
		t.Fatal(err)
	}

	// Only 2 should remain.
	if !reflect.DeepEqual(r.attPool.AggregatedAttestations(), []*ethpb.Attestation{att2}) {
		t.Error("Did not get wanted attestation from pool")
	}
}

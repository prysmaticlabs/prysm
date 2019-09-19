package sync

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
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

	s := &pb.BeaconState{FinalizedCheckpoint: &ethpb.Checkpoint{Epoch: 1}}
	parent := &ethpb.BeaconBlock{}
	if err := db.SaveBlock(context.Background(), parent); err != nil {
		t.Fatal(err)
	}
	parentRoot, _ := ssz.SigningRoot(parent)
	r := &RegularSync{
		db:    db,
		chain: &mock.ChainService{State: s},
	}

	b := &ethpb.BeaconBlock{Slot: 1, ParentRoot: parentRoot[:]}
	if err := r.beaconBlockSubscriber(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	testutil.AssertLogsContain(t, hook, "Received a block that's older than finalized checkpoint, 1 < 64")

	hook.Reset()
	b.Slot = params.BeaconConfig().SlotsPerEpoch
	if err := r.beaconBlockSubscriber(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	testutil.AssertLogsDoNotContain(t, hook, "Received a block that's older than finalized checkpoint")
}

package aggregator

import (
	"context"
	"crypto/rand"
	"strings"
	"testing"

	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func init() {
	// Use minimal config to reduce test setup time.
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
}

func TestSubmitAggregateAndProof_Syncing(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()

	s := &pbp2p.BeaconState{}

	aggregatorServer := &Server{
		HeadFetcher: &mock.ChainService{State: s},
		SyncChecker: &mockSync.Sync{IsSyncing: true},
		BeaconDB:    db,
	}

	req := &pb.AggregationRequest{CommitteeIndex: 1}
	wanted := "Syncing to latest head, not ready to respond"
	if _, err := aggregatorServer.SubmitAggregateAndProof(ctx, req); !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestSubmitAggregateAndProof_CantFindValidatorIndex(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()

	s := &pbp2p.BeaconState{
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	aggregatorServer := &Server{
		HeadFetcher: &mock.ChainService{State: s},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
		BeaconDB:    db,
	}

	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sig := priv.Sign([]byte{'A'}, 0)
	req := &pb.AggregationRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal()}
	wanted := "could not locate validator index in DB"
	if _, err := aggregatorServer.SubmitAggregateAndProof(ctx, req); !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestSubmitAggregateAndProof_IsAggregator(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()

	s := &pbp2p.BeaconState{
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	aggregatorServer := &Server{
		HeadFetcher: &mock.ChainService{State: s},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
		BeaconDB:    db,
	}

	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sig := priv.Sign([]byte{'A'}, 0)
	pubKey := [48]byte{'A'}
	req := &pb.AggregationRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey[:]}
	if err := aggregatorServer.BeaconDB.SaveValidatorIndex(ctx, pubKey, 100); err != nil {
		t.Fatal(err)
	}

	res, err := aggregatorServer.SubmitAggregateAndProof(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	if !res.Aggregated {
		t.Error("Wanted aggregator true, got false")
	}
}

func TestSubmitAggregateAndProof_IsNotAggregator(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()
	deposits, _, privKeys := testutil.SetupInitialDeposits(t, 2048)
	s, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}

	aggregatorServer := &Server{
		HeadFetcher: &mock.ChainService{State: s},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
		BeaconDB:    db,
	}

	sig := privKeys[0].Sign([]byte{}, 0)
	pubKey := bytesutil.ToBytes48(privKeys[0].PublicKey().Marshal())
	req := &pb.AggregationRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey[:]}
	if err := aggregatorServer.BeaconDB.SaveValidatorIndex(ctx, pubKey, 100); err != nil {
		t.Fatal(err)
	}

	res, err := aggregatorServer.SubmitAggregateAndProof(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	if res.Aggregated {
		t.Error("Wanted aggregator false, got true")
	}
}

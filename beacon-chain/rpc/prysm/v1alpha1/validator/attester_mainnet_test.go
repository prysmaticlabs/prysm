package validator

import (
	"context"
	"testing"
	"time"

	mock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	mockp2p "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/v3/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"google.golang.org/protobuf/proto"
)

func TestAttestationDataAtSlot_HandlesFarAwayJustifiedEpoch(t *testing.T) {
	// Scenario:
	//
	// State slot = 10000
	// Last justified slot = epoch start of 1500
	// HistoricalRootsLimit = 8192
	//
	// More background: https://github.com/prysmaticlabs/prysm/issues/2153
	// This test breaks if it doesn't use mainnet config

	// Ensure HistoricalRootsLimit matches scenario
	params.SetupTestConfigCleanup(t)
	cfg := params.MainnetConfig().Copy()
	cfg.HistoricalRootsLimit = 8192
	params.OverrideBeaconConfig(cfg)

	block := util.NewBeaconBlock()
	block.Block.Slot = 10000
	epochBoundaryBlock := util.NewBeaconBlock()
	var err error
	epochBoundaryBlock.Block.Slot, err = slots.EpochStart(slots.ToEpoch(10000))
	require.NoError(t, err)
	justifiedBlock := util.NewBeaconBlock()
	justifiedBlock.Block.Slot, err = slots.EpochStart(slots.ToEpoch(1500))
	require.NoError(t, err)
	justifiedBlock.Block.Slot -= 2 // Imagine two skip block
	blockRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not hash beacon block")
	justifiedBlockRoot, err := justifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not hash justified block")
	epochBoundaryRoot, err := epochBoundaryBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not hash justified block")
	slot := primitives.Slot(10000)

	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(slot))
	err = beaconState.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: slots.ToEpoch(1500),
		Root:  justifiedBlockRoot[:],
	})
	require.NoError(t, err)
	blockRoots := beaconState.BlockRoots()
	blockRoots[1] = blockRoot[:]
	blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = epochBoundaryRoot[:]
	blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedBlockRoot[:]
	require.NoError(t, beaconState.SetBlockRoots(blockRoots))
	chainService := &mock.ChainService{
		Genesis: time.Now(),
	}
	offset := int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
	attesterServer := &Server{
		P2P:              &mockp2p.MockBroadcaster{},
		AttestationCache: cache.NewAttestationCache(),
		HeadFetcher:      &mock.ChainService{State: beaconState, Root: blockRoot[:]},
		FinalizationFetcher: &mock.ChainService{
			CurrentJustifiedCheckPoint: beaconState.CurrentJustifiedCheckpoint(),
		},
		SyncChecker:   &mockSync.Sync{IsSyncing: false},
		TimeFetcher:   &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*offset) * time.Second)},
		StateNotifier: chainService.StateNotifier(),
	}

	req := &ethpb.AttestationDataRequest{
		CommitteeIndex: 0,
		Slot:           10000,
	}
	res, err := attesterServer.GetAttestationData(context.Background(), req)
	require.NoError(t, err, "Could not get attestation info at slot")

	expectedInfo := &ethpb.AttestationData{
		Slot:            req.Slot,
		BeaconBlockRoot: blockRoot[:],
		Source: &ethpb.Checkpoint{
			Epoch: slots.ToEpoch(1500),
			Root:  justifiedBlockRoot[:],
		},
		Target: &ethpb.Checkpoint{
			Epoch: 312,
			Root:  blockRoot[:],
		},
	}

	if !proto.Equal(res, expectedInfo) {
		t.Errorf("Expected attestation info to match, received %v, wanted %v", res, expectedInfo)
	}
}

package validator

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateV0"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func Test_DetectDoppelganger_NoHead(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	// Set head state to nil
	chainService := &mockChain.ChainService{State: nil}
	vs := &Server{
		Ctx: ctx,
		ChainStartFetcher: &mockPOW.POWChain{
			ChainFeed: new(event.Feed),
		},
		BeaconDB:      beaconDB,
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
	}

	req := &ethpb.DetectDoppelgangerRequest{
		PubKeysTargets: nil,
	}
	_, err := vs.DetectDoppelganger(ctx, req)
	assert.ErrorContains(t, "Doppelganger rpc service - Could not get head state", err)
}

func Test_DetectDoppelganger_TargetHeadClose(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	priv1, err := bls.RandKey()
	require.NoError(t, err)

	pubKey1 := priv1.PublicKey().Marshal()
	slot := types.Slot(4000)
	beaconState := &pbp2p.BeaconState{
		Slot: slot,
		Validators: []*ethpb.Validator{
			{
				ActivationEpoch:       0,
				ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
				PublicKey:             pubKey1,
				WithdrawalCredentials: make([]byte, 32),
			},
		},
	}

	block := testutil.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err)
	// Set head state to nil
	trie, err := stateV0.InitializeFromProtoUnsafe(beaconState)
	require.NoError(t, err)
	vs := &Server{
		BeaconDB:           beaconDB,
		Ctx:                ctx,
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		HeadFetcher:        &mockChain.ChainService{State: trie, Root: genesisRoot[:]},
	}

	pKT := make([]*ethpb.PubKeyTarget, 0)

	// Use the same slot so that Head - Target is less than N(=2) Epochs aparts.
	pKT = append(pKT, &ethpb.PubKeyTarget{PubKey: pubKey1,
		TargetEpoch: types.Epoch(slot / params.BeaconConfig().SlotsPerEpoch)})
	req := &ethpb.DetectDoppelgangerRequest{
		PubKeysTargets: pKT}
	_, err = vs.DetectDoppelganger(ctx, req)
	assert.NoError(t, err)
	//assert.ErrorContains(t,"Doppelganger rpc service - Could not get previous state root",err)
}

func Test_DetectDoppelganger_NoPrevState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	priv1, err := bls.RandKey()
	require.NoError(t, err)

	pubKey1 := priv1.PublicKey().Marshal()
	slot := types.Slot(4000)
	beaconState := &pbp2p.BeaconState{
		Slot: slot,
		Validators: []*ethpb.Validator{
			{
				ActivationEpoch:       0,
				ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
				PublicKey:             pubKey1,
				WithdrawalCredentials: make([]byte, 32),
			},
		},
	}

	block := testutil.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err)
	// Set head state to nil
	trie, err := stateV0.InitializeFromProtoUnsafe(beaconState)
	require.NoError(t, err)
	vs := &Server{
		BeaconDB:           beaconDB,
		Ctx:                ctx,
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		HeadFetcher:        &mockChain.ChainService{State: trie, Root: genesisRoot[:]},
	}

	pKT := make([]*ethpb.PubKeyTarget, 0)

	// Use the same slot so that Head - Target is less than N(=2) Epochs aparts.
	pKT = append(pKT, &ethpb.PubKeyTarget{PubKey: pubKey1,
		TargetEpoch: types.Epoch(slot.Sub(20) / params.BeaconConfig().SlotsPerEpoch)})
	req := &ethpb.DetectDoppelgangerRequest{
		PubKeysTargets: pKT}
	_, err = vs.DetectDoppelganger(ctx, req)
	assert.ErrorContains(t, "Doppelganger rpc service - Could not get previous state root", err)
}

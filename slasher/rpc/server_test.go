package rpc

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateV0"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/p2putils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/slasher/beaconclient"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	"github.com/prysmaticlabs/prysm/slasher/detection"
)

func TestServer_IsSlashableAttestation(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	bClient := mock.NewMockBeaconChainClient(ctrl)
	nClient := mock.NewMockNodeClient(ctrl)
	ctx := context.Background()

	_, keys, err := testutil.DeterministicDepositsAndKeys(4)
	require.NoError(t, err)
	wantedValidators1 := &ethpb.Validators{
		ValidatorList: []*ethpb.Validators_ValidatorContainer{
			{
				Index: 3, Validator: &ethpb.Validator{PublicKey: keys[3].PublicKey().Marshal()},
			},
		},
	}

	wantedGenesis := &ethpb.Genesis{
		GenesisValidatorsRoot: bytesutil.PadTo([]byte("I am genesis"), 32),
	}

	savedAttestation := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{3},
		Data: &ethpb.AttestationData{
			Source:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
			Target:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
			BeaconBlockRoot: make([]byte, 32),
		},
		Signature: make([]byte, 96),
	}

	cfg := &detection.Config{
		SlasherDB: db,
	}
	fork, err := p2putils.Fork(savedAttestation.Data.Target.Epoch)
	require.NoError(t, err)

	bcCfg := &beaconclient.Config{BeaconClient: bClient, NodeClient: nClient, SlasherDB: db}
	bs, err := beaconclient.NewService(ctx, bcCfg)
	require.NoError(t, err)
	ds := detection.NewService(ctx, cfg)
	server := Server{ctx: ctx, detector: ds, slasherDB: db, beaconClient: bs}
	nClient.EXPECT().GetGenesis(gomock.Any(), gomock.Any()).Return(wantedGenesis, nil).AnyTimes()
	bClient.EXPECT().ListValidators(
		gomock.Any(),
		gomock.Any(),
	).Return(wantedValidators1, nil).AnyTimes()
	domain, err := helpers.Domain(fork, savedAttestation.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester, wantedGenesis.GenesisValidatorsRoot)
	require.NoError(t, err)
	wg := sync.WaitGroup{}
	wg.Add(100)
	var wentThrough bool
	for i := types.Slot(0); i < 100; i++ {
		go func(j types.Slot) {
			defer wg.Done()
			iatt := stateV0.CopyIndexedAttestation(savedAttestation)
			iatt.Data.Slot += j
			root, err := helpers.ComputeSigningRoot(iatt.Data, domain)
			require.NoError(t, err)
			validatorSig := keys[iatt.AttestingIndices[0]].Sign(root[:])
			marshalledSig := validatorSig.Marshal()
			iatt.Signature = marshalledSig
			slashings, err := server.IsSlashableAttestation(ctx, iatt)
			require.NoError(t, err, "Got error while trying to detect slashing")

			if len(slashings.AttesterSlashing) == 0 && !wentThrough {
				wentThrough = true
			} else if len(slashings.AttesterSlashing) == 0 && wentThrough {
				t.Fatalf("Only one attestation should go through without slashing: %v", iatt)
			}
		}(i)
	}
	wg.Wait()

}

func TestServer_IsSlashableAttestationNoUpdate(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	bClient := mock.NewMockBeaconChainClient(ctrl)
	nClient := mock.NewMockNodeClient(ctrl)
	ctx := context.Background()

	_, keys, err := testutil.DeterministicDepositsAndKeys(4)
	require.NoError(t, err)
	wantedValidators1 := &ethpb.Validators{
		ValidatorList: []*ethpb.Validators_ValidatorContainer{
			{
				Index: 3, Validator: &ethpb.Validator{PublicKey: keys[3].PublicKey().Marshal()},
			},
		},
	}
	bClient.EXPECT().ListValidators(
		gomock.Any(),
		gomock.Any(),
	).Return(wantedValidators1, nil)

	wantedGenesis := &ethpb.Genesis{
		GenesisValidatorsRoot: bytesutil.PadTo([]byte("I am genesis"), 32),
	}
	nClient.EXPECT().GetGenesis(gomock.Any(), gomock.Any()).Return(wantedGenesis, nil)
	savedAttestation := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{3},
		Data: &ethpb.AttestationData{
			Source:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
			Target:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
			BeaconBlockRoot: make([]byte, 32),
		},
		Signature: make([]byte, 96),
	}
	incomingAtt := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 3},
		Data: &ethpb.AttestationData{
			Source:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
			Target:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
			BeaconBlockRoot: make([]byte, 32),
		},
		Signature: make([]byte, 96),
	}
	cfg := &detection.Config{
		SlasherDB: db,
	}
	fork, err := p2putils.Fork(savedAttestation.Data.Target.Epoch)
	require.NoError(t, err)
	domain, err := helpers.Domain(fork, savedAttestation.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester, wantedGenesis.GenesisValidatorsRoot)
	require.NoError(t, err)
	root, err := helpers.ComputeSigningRoot(savedAttestation.Data, domain)
	require.NoError(t, err)
	var sig []bls.Signature
	for _, idx := range savedAttestation.AttestingIndices {
		validatorSig := keys[idx].Sign(root[:])
		sig = append(sig, validatorSig)
	}
	aggSig := bls.AggregateSignatures(sig)
	marshalledSig := aggSig.Marshal()

	savedAttestation.Signature = marshalledSig

	bcCfg := &beaconclient.Config{BeaconClient: bClient, NodeClient: nClient, SlasherDB: db}
	bs, err := beaconclient.NewService(ctx, bcCfg)
	require.NoError(t, err)
	ds := detection.NewService(ctx, cfg)
	server := Server{ctx: ctx, detector: ds, slasherDB: db, beaconClient: bs}
	slashings, err := server.IsSlashableAttestation(ctx, savedAttestation)
	require.NoError(t, err, "Got error while trying to detect slashing")
	require.Equal(t, 0, len(slashings.AttesterSlashing), "Found slashings while no slashing should have been found on first attestation")
	sl, err := server.IsSlashableAttestationNoUpdate(ctx, incomingAtt)
	require.NoError(t, err, "Got error while trying to detect slashing")
	require.Equal(t, true, sl.Slashable, "Attestation should be found to be slashable")
}

func TestServer_IsSlashableBlock(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	bClient := mock.NewMockBeaconChainClient(ctrl)
	nClient := mock.NewMockNodeClient(ctrl)
	ctx := context.Background()

	_, keys, err := testutil.DeterministicDepositsAndKeys(4)
	require.NoError(t, err)
	wantedValidators := &ethpb.Validators{
		ValidatorList: []*ethpb.Validators_ValidatorContainer{
			{
				Index: 1, Validator: &ethpb.Validator{PublicKey: keys[1].PublicKey().Marshal()},
			},
		},
	}
	bClient.EXPECT().ListValidators(
		gomock.Any(),
		gomock.Any(),
	).Return(wantedValidators, nil).AnyTimes()

	wantedGenesis := &ethpb.Genesis{
		GenesisValidatorsRoot: bytesutil.PadTo([]byte("I am genesis"), 32),
	}
	nClient.EXPECT().GetGenesis(gomock.Any(), gomock.Any()).Return(wantedGenesis, nil).AnyTimes()
	savedBlock := testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:          1,
			ProposerIndex: 1,
			BodyRoot:      bytesutil.PadTo([]byte("body root"), 32),
		},
	})

	cfg := &detection.Config{
		SlasherDB: db,
	}
	savedBlockEpoch := helpers.SlotToEpoch(savedBlock.Header.Slot)
	fork, err := p2putils.Fork(savedBlockEpoch)
	require.NoError(t, err)
	domain, err := helpers.Domain(fork, savedBlockEpoch, params.BeaconConfig().DomainBeaconProposer, wantedGenesis.GenesisValidatorsRoot)
	require.NoError(t, err)

	bcCfg := &beaconclient.Config{BeaconClient: bClient, NodeClient: nClient, SlasherDB: db}
	bs, err := beaconclient.NewService(ctx, bcCfg)
	require.NoError(t, err)
	ds := detection.NewService(ctx, cfg)
	server := Server{ctx: ctx, detector: ds, slasherDB: db, beaconClient: bs}

	wg := sync.WaitGroup{}
	wg.Add(100)
	var wentThrough bool
	for i := uint64(0); i < 100; i++ {
		go func(j uint64) {
			defer wg.Done()
			sbbh := stateV0.CopySignedBeaconBlockHeader(savedBlock)
			sbbh.Header.BodyRoot = bytesutil.PadTo([]byte(fmt.Sprintf("%d", j)), 32)
			bhr, err := sbbh.Header.HashTreeRoot()
			assert.NoError(t, err)
			sszBytes := p2ptypes.SSZBytes(bhr[:])
			root, err := helpers.ComputeSigningRoot(&sszBytes, domain)
			assert.NoError(t, err)
			sbbh.Signature = keys[sbbh.Header.ProposerIndex].Sign(root[:]).Marshal()
			slashings, err := server.IsSlashableBlock(ctx, sbbh)
			require.NoError(t, err, "Got error while trying to detect slashing")
			if len(slashings.ProposerSlashing) == 0 && !wentThrough {
				wentThrough = true
			} else if len(slashings.ProposerSlashing) == 0 && wentThrough {
				t.Fatalf("Only one block should go through without slashing: %v", sbbh)
			}
		}(i)
	}
	wg.Wait()
}

func TestServer_IsSlashableBlockNoUpdate(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	bClient := mock.NewMockBeaconChainClient(ctrl)
	nClient := mock.NewMockNodeClient(ctrl)
	ctx := context.Background()

	_, keys, err := testutil.DeterministicDepositsAndKeys(4)
	require.NoError(t, err)
	wantedValidators := &ethpb.Validators{
		ValidatorList: []*ethpb.Validators_ValidatorContainer{
			{
				Index: 1, Validator: &ethpb.Validator{PublicKey: keys[1].PublicKey().Marshal()},
			},
		},
	}
	bClient.EXPECT().ListValidators(
		gomock.Any(),
		gomock.Any(),
	).Return(wantedValidators, nil)

	wantedGenesis := &ethpb.Genesis{
		GenesisValidatorsRoot: bytesutil.PadTo([]byte("I am genesis"), 32),
	}
	nClient.EXPECT().GetGenesis(gomock.Any(), gomock.Any()).Return(wantedGenesis, nil)
	savedBlock := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:          1,
			ProposerIndex: 1,
			BodyRoot:      bytesutil.PadTo([]byte("body root"), 32),
			StateRoot:     bytesutil.PadTo([]byte("state root"), 32),
			ParentRoot:    bytesutil.PadTo([]byte("parent root"), 32),
		},
		Signature: make([]byte, 96),
	}
	incomingBlock := &ethpb.BeaconBlockHeader{
		Slot:          1,
		ProposerIndex: 1,
		BodyRoot:      bytesutil.PadTo([]byte("body root2"), 32),
		StateRoot:     bytesutil.PadTo([]byte("state root2"), 32),
		ParentRoot:    bytesutil.PadTo([]byte("parent root2"), 32),
	}
	cfg := &detection.Config{
		SlasherDB: db,
	}
	savedBlockEpoch := helpers.SlotToEpoch(savedBlock.Header.Slot)
	fork, err := p2putils.Fork(savedBlockEpoch)
	require.NoError(t, err)
	domain, err := helpers.Domain(fork, savedBlockEpoch, params.BeaconConfig().DomainBeaconProposer, wantedGenesis.GenesisValidatorsRoot)
	require.NoError(t, err)
	bhr, err := savedBlock.Header.HashTreeRoot()
	require.NoError(t, err)
	sszBytes := p2ptypes.SSZBytes(bhr[:])
	root, err := helpers.ComputeSigningRoot(&sszBytes, domain)
	require.NoError(t, err)
	blockSig := keys[savedBlock.Header.ProposerIndex].Sign(root[:])
	marshalledSig := blockSig.Marshal()
	savedBlock.Signature = marshalledSig
	bcCfg := &beaconclient.Config{BeaconClient: bClient, NodeClient: nClient, SlasherDB: db}
	bs, err := beaconclient.NewService(ctx, bcCfg)
	require.NoError(t, err)
	ds := detection.NewService(ctx, cfg)
	server := Server{ctx: ctx, detector: ds, slasherDB: db, beaconClient: bs}
	slashings, err := server.IsSlashableBlock(ctx, savedBlock)
	require.NoError(t, err, "Got error while trying to detect slashing")
	require.Equal(t, 0, len(slashings.ProposerSlashing), "Found slashings while no slashing should have been found on first block")
	sl, err := server.IsSlashableBlockNoUpdate(ctx, incomingBlock)
	require.NoError(t, err, "Got error while trying to detect slashing")
	require.Equal(t, true, sl.Slashable, "Block should be found to be slashable")
}

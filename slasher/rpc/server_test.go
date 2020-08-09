package rpc

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/p2putils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
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
	if err != nil {
		t.Fatal(err)
	}
	wantedValidators1 := &ethpb.Validators{
		ValidatorList: []*ethpb.Validators_ValidatorContainer{
			{
				Index: 3, Validator: &ethpb.Validator{PublicKey: keys[3].PublicKey().Marshal()},
			},
		},
	}

	wantedGenesis := &ethpb.Genesis{
		GenesisValidatorsRoot: []byte("I am genesis"),
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
	if err != nil {
		t.Fatal(err)
	}

	bcCfg := &beaconclient.Config{BeaconClient: bClient, NodeClient: nClient, SlasherDB: db}
	bs, err := beaconclient.NewBeaconClientService(ctx, bcCfg)
	ds := detection.NewDetectionService(ctx, cfg)
	server := Server{ctx: ctx, detector: ds, slasherDB: db, beaconClient: bs}
	nClient.EXPECT().GetGenesis(gomock.Any(), gomock.Any()).Return(wantedGenesis, nil).AnyTimes()
	bClient.EXPECT().ListValidators(
		gomock.Any(),
		gomock.Any(),
	).Return(wantedValidators1, nil).AnyTimes()
	domain, err := helpers.Domain(fork, savedAttestation.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester, wantedGenesis.GenesisValidatorsRoot)
	if err != nil {
		t.Fatal(err)
	}
	wg := sync.WaitGroup{}
	wg.Add(100)
	var wentThrough bool
	for i := uint64(0); i < 100; i++ {
		go func(j uint64) {
			defer wg.Done()
			iatt := state.CopyIndexedAttestation(savedAttestation)
			iatt.Data.Slot += j
			root, err := helpers.ComputeSigningRoot(iatt.Data, domain)
			if err != nil {
				t.Error(err)
			}
			var validatorSig bls.Signature
			validatorSig = keys[iatt.AttestingIndices[0]].Sign(root[:])
			marshalledSig := validatorSig.Marshal()
			iatt.Signature = marshalledSig
			slashings, err := server.IsSlashableAttestation(ctx, iatt)
			if err != nil {
				t.Fatalf("got error while trying to detect slashing: %v", err)
			}

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
	if err != nil {
		t.Fatal(err)
	}
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
		GenesisValidatorsRoot: []byte("I am genesis"),
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
	if err != nil {
		t.Fatal(err)
	}
	domain, err := helpers.Domain(fork, savedAttestation.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester, wantedGenesis.GenesisValidatorsRoot)
	if err != nil {
		t.Fatal(err)
	}
	root, err := helpers.ComputeSigningRoot(savedAttestation.Data, domain)
	if err != nil {
		t.Error(err)
	}
	sig := []bls.Signature{}
	for _, idx := range savedAttestation.AttestingIndices {
		validatorSig := keys[idx].Sign(root[:])
		sig = append(sig, validatorSig)
	}
	aggSig := bls.AggregateSignatures(sig)
	marshalledSig := aggSig.Marshal()

	savedAttestation.Signature = marshalledSig

	bcCfg := &beaconclient.Config{BeaconClient: bClient, NodeClient: nClient, SlasherDB: db}
	bs, err := beaconclient.NewBeaconClientService(ctx, bcCfg)
	ds := detection.NewDetectionService(ctx, cfg)
	server := Server{ctx: ctx, detector: ds, slasherDB: db, beaconClient: bs}
	slashings, err := server.IsSlashableAttestation(ctx, savedAttestation)
	if err != nil {
		t.Fatalf("got error while trying to detect slashing: %v", err)
	}
	if len(slashings.AttesterSlashing) != 0 {
		t.Fatalf("Found slashings while no slashing should have been found on first attestation: %v slashing found: %v", savedAttestation, slashings)
	}
	sl, err := server.IsSlashableAttestationNoUpdate(ctx, incomingAtt)
	if err != nil {
		t.Fatalf("got error while trying to detect slashing: %v", err)
	}
	if sl.Slashable != true {
		t.Fatalf("attestation should be found to be slashable. got: %v", sl.Slashable)
	}
}

func TestServer_IsSlashableBlock(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	bClient := mock.NewMockBeaconChainClient(ctrl)
	nClient := mock.NewMockNodeClient(ctrl)
	ctx := context.Background()

	_, keys, err := testutil.DeterministicDepositsAndKeys(4)
	if err != nil {
		t.Fatal(err)
	}
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
		GenesisValidatorsRoot: []byte("I am genesis"),
	}
	nClient.EXPECT().GetGenesis(gomock.Any(), gomock.Any()).Return(wantedGenesis, nil).AnyTimes()
	savedBlock := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:          1,
			ProposerIndex: 1,
			BodyRoot:      bytesutil.PadTo([]byte("body root"), 32),
			StateRoot:     make([]byte, 32),
			ParentRoot:    make([]byte, 32),
		},
		Signature: make([]byte, 96),
	}

	cfg := &detection.Config{
		SlasherDB: db,
	}
	savedBlockEpoch := helpers.SlotToEpoch(savedBlock.Header.Slot)
	fork, err := p2putils.Fork(savedBlockEpoch)
	if err != nil {
		t.Fatal(err)
	}
	domain, err := helpers.Domain(fork, savedBlockEpoch, params.BeaconConfig().DomainBeaconProposer, wantedGenesis.GenesisValidatorsRoot)
	if err != nil {
		t.Fatal(err)
	}

	bcCfg := &beaconclient.Config{BeaconClient: bClient, NodeClient: nClient, SlasherDB: db}
	bs, err := beaconclient.NewBeaconClientService(ctx, bcCfg)
	ds := detection.NewDetectionService(ctx, cfg)
	server := Server{ctx: ctx, detector: ds, slasherDB: db, beaconClient: bs}

	wg := sync.WaitGroup{}
	wg.Add(100)
	var wentThrough bool
	for i := uint64(0); i < 100; i++ {
		go func(j uint64) {
			defer wg.Done()
			sbbh := state.CopySignedBeaconBlockHeader(savedBlock)
			sbbh.Header.BodyRoot = bytesutil.PadTo([]byte(fmt.Sprintf("%d", j)), 32)
			bhr, err := stateutil.BlockHeaderRoot(sbbh.Header)
			if err != nil {
				t.Error(err)
			}
			root, err := helpers.ComputeSigningRoot(bhr, domain)
			if err != nil {
				t.Error(err)
			}
			sbbh.Signature = keys[sbbh.Header.ProposerIndex].Sign(root[:]).Marshal()
			slashings, err := server.IsSlashableBlock(ctx, sbbh)
			if err != nil {
				t.Fatalf("got error while trying to detect slashing: %v", err)
			}
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
	if err != nil {
		t.Fatal(err)
	}
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
		GenesisValidatorsRoot: []byte("I am genesis"),
	}
	nClient.EXPECT().GetGenesis(gomock.Any(), gomock.Any()).Return(wantedGenesis, nil)
	savedBlock := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:          1,
			ProposerIndex: 1,
			BodyRoot:      bytesutil.PadTo([]byte("body root"), 32),
		},
	}
	incomingBlock := &ethpb.BeaconBlockHeader{
		Slot:          1,
		ProposerIndex: 1,
		BodyRoot:      bytesutil.PadTo([]byte("body root2"), 32),
	}
	cfg := &detection.Config{
		SlasherDB: db,
	}
	savedBlockEpoch := helpers.SlotToEpoch(savedBlock.Header.Slot)
	fork, err := p2putils.Fork(savedBlockEpoch)
	if err != nil {
		t.Fatal(err)
	}
	domain, err := helpers.Domain(fork, savedBlockEpoch, params.BeaconConfig().DomainBeaconProposer, wantedGenesis.GenesisValidatorsRoot)
	if err != nil {
		t.Fatal(err)
	}
	bhr, err := stateutil.BlockHeaderRoot(savedBlock.Header)
	if err != nil {
		t.Error(err)
	}
	root, err := helpers.ComputeSigningRoot(bhr, domain)
	if err != nil {
		t.Error(err)
	}
	blockSig := keys[savedBlock.Header.ProposerIndex].Sign(root[:])
	marshalledSig := blockSig.Marshal()
	savedBlock.Signature = marshalledSig
	bcCfg := &beaconclient.Config{BeaconClient: bClient, NodeClient: nClient, SlasherDB: db}
	bs, err := beaconclient.NewBeaconClientService(ctx, bcCfg)
	ds := detection.NewDetectionService(ctx, cfg)
	server := Server{ctx: ctx, detector: ds, slasherDB: db, beaconClient: bs}
	slashings, err := server.IsSlashableBlock(ctx, savedBlock)
	if err != nil {
		t.Fatalf("got error while trying to detect slashing: %v", err)
	}
	if len(slashings.ProposerSlashing) != 0 {
		t.Fatalf("Found slashings while no slashing should have been found on first block: %v slashing found: %v", savedBlock, slashings)
	}
	sl, err := server.IsSlashableBlockNoUpdate(ctx, incomingBlock)
	if err != nil {
		t.Fatalf("got error while trying to detect slashing: %v", err)
	}
	if sl.Slashable != true {
		t.Fatalf("block should be found to be slashable. got: %v", sl.Slashable)
	}
}

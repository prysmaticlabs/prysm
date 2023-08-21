package beacon

import (
	"context"
	"testing"
	"time"

	blockchainmock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	prysmtime "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/blstoexec"
	blstoexecmock "github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/blstoexec/mock"
	slashingsmock "github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/slashings/mock"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/voluntaryexits/mock"
	p2pMock "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	state_native "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/common"
	"github.com/prysmaticlabs/prysm/v4/crypto/hash"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/encoding/ssz"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	ethpbv1alpha1 "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestListPoolAttesterSlashings(t *testing.T) {
	bs, err := util.NewBeaconState()
	require.NoError(t, err)
	slashing1 := &ethpbv1alpha1.AttesterSlashing{
		Attestation_1: &ethpbv1alpha1.IndexedAttestation{
			AttestingIndices: []uint64{1, 10},
			Data: &ethpbv1alpha1.AttestationData{
				Slot:            1,
				CommitteeIndex:  1,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot1"), 32),
				Source: &ethpbv1alpha1.Checkpoint{
					Epoch: 1,
					Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
				},
				Target: &ethpbv1alpha1.Checkpoint{
					Epoch: 10,
					Root:  bytesutil.PadTo([]byte("targetroot1"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature1"), 96),
		},
		Attestation_2: &ethpbv1alpha1.IndexedAttestation{
			AttestingIndices: []uint64{2, 20},
			Data: &ethpbv1alpha1.AttestationData{
				Slot:            2,
				CommitteeIndex:  2,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot2"), 32),
				Source: &ethpbv1alpha1.Checkpoint{
					Epoch: 2,
					Root:  bytesutil.PadTo([]byte("sourceroot2"), 32),
				},
				Target: &ethpbv1alpha1.Checkpoint{
					Epoch: 20,
					Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature2"), 96),
		},
	}
	slashing2 := &ethpbv1alpha1.AttesterSlashing{
		Attestation_1: &ethpbv1alpha1.IndexedAttestation{
			AttestingIndices: []uint64{3, 30},
			Data: &ethpbv1alpha1.AttestationData{
				Slot:            3,
				CommitteeIndex:  3,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot3"), 32),
				Source: &ethpbv1alpha1.Checkpoint{
					Epoch: 3,
					Root:  bytesutil.PadTo([]byte("sourceroot3"), 32),
				},
				Target: &ethpbv1alpha1.Checkpoint{
					Epoch: 30,
					Root:  bytesutil.PadTo([]byte("targetroot3"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature3"), 96),
		},
		Attestation_2: &ethpbv1alpha1.IndexedAttestation{
			AttestingIndices: []uint64{4, 40},
			Data: &ethpbv1alpha1.AttestationData{
				Slot:            4,
				CommitteeIndex:  4,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot4"), 32),
				Source: &ethpbv1alpha1.Checkpoint{
					Epoch: 4,
					Root:  bytesutil.PadTo([]byte("sourceroot4"), 32),
				},
				Target: &ethpbv1alpha1.Checkpoint{
					Epoch: 40,
					Root:  bytesutil.PadTo([]byte("targetroot4"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature4"), 96),
		},
	}

	s := &Server{
		ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
		SlashingsPool:    &slashingsmock.PoolMock{PendingAttSlashings: []*ethpbv1alpha1.AttesterSlashing{slashing1, slashing2}},
	}

	resp, err := s.ListPoolAttesterSlashings(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, 2, len(resp.Data))
	assert.DeepEqual(t, migration.V1Alpha1AttSlashingToV1(slashing1), resp.Data[0])
	assert.DeepEqual(t, migration.V1Alpha1AttSlashingToV1(slashing2), resp.Data[1])
}

func TestListPoolProposerSlashings(t *testing.T) {
	bs, err := util.NewBeaconState()
	require.NoError(t, err)
	slashing1 := &ethpbv1alpha1.ProposerSlashing{
		Header_1: &ethpbv1alpha1.SignedBeaconBlockHeader{
			Header: &ethpbv1alpha1.BeaconBlockHeader{
				Slot:          1,
				ProposerIndex: 1,
				ParentRoot:    bytesutil.PadTo([]byte("parentroot1"), 32),
				StateRoot:     bytesutil.PadTo([]byte("stateroot1"), 32),
				BodyRoot:      bytesutil.PadTo([]byte("bodyroot1"), 32),
			},
			Signature: bytesutil.PadTo([]byte("signature1"), 96),
		},
		Header_2: &ethpbv1alpha1.SignedBeaconBlockHeader{
			Header: &ethpbv1alpha1.BeaconBlockHeader{
				Slot:          2,
				ProposerIndex: 2,
				ParentRoot:    bytesutil.PadTo([]byte("parentroot2"), 32),
				StateRoot:     bytesutil.PadTo([]byte("stateroot2"), 32),
				BodyRoot:      bytesutil.PadTo([]byte("bodyroot2"), 32),
			},
			Signature: bytesutil.PadTo([]byte("signature2"), 96),
		},
	}
	slashing2 := &ethpbv1alpha1.ProposerSlashing{
		Header_1: &ethpbv1alpha1.SignedBeaconBlockHeader{
			Header: &ethpbv1alpha1.BeaconBlockHeader{
				Slot:          3,
				ProposerIndex: 3,
				ParentRoot:    bytesutil.PadTo([]byte("parentroot3"), 32),
				StateRoot:     bytesutil.PadTo([]byte("stateroot3"), 32),
				BodyRoot:      bytesutil.PadTo([]byte("bodyroot3"), 32),
			},
			Signature: bytesutil.PadTo([]byte("signature3"), 96),
		},
		Header_2: &ethpbv1alpha1.SignedBeaconBlockHeader{
			Header: &ethpbv1alpha1.BeaconBlockHeader{
				Slot:          4,
				ProposerIndex: 4,
				ParentRoot:    bytesutil.PadTo([]byte("parentroot4"), 32),
				StateRoot:     bytesutil.PadTo([]byte("stateroot4"), 32),
				BodyRoot:      bytesutil.PadTo([]byte("bodyroot4"), 32),
			},
			Signature: bytesutil.PadTo([]byte("signature4"), 96),
		},
	}

	s := &Server{
		ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
		SlashingsPool:    &slashingsmock.PoolMock{PendingPropSlashings: []*ethpbv1alpha1.ProposerSlashing{slashing1, slashing2}},
	}

	resp, err := s.ListPoolProposerSlashings(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, 2, len(resp.Data))
	assert.DeepEqual(t, migration.V1Alpha1ProposerSlashingToV1(slashing1), resp.Data[0])
	assert.DeepEqual(t, migration.V1Alpha1ProposerSlashingToV1(slashing2), resp.Data[1])
}

func TestListPoolVoluntaryExits(t *testing.T) {
	bs, err := util.NewBeaconState()
	require.NoError(t, err)
	exit1 := &ethpbv1alpha1.SignedVoluntaryExit{
		Exit: &ethpbv1alpha1.VoluntaryExit{
			Epoch:          1,
			ValidatorIndex: 1,
		},
		Signature: bytesutil.PadTo([]byte("signature1"), 96),
	}
	exit2 := &ethpbv1alpha1.SignedVoluntaryExit{
		Exit: &ethpbv1alpha1.VoluntaryExit{
			Epoch:          2,
			ValidatorIndex: 2,
		},
		Signature: bytesutil.PadTo([]byte("signature2"), 96),
	}

	s := &Server{
		ChainInfoFetcher:   &blockchainmock.ChainService{State: bs},
		VoluntaryExitsPool: &mock.PoolMock{Exits: []*ethpbv1alpha1.SignedVoluntaryExit{exit1, exit2}},
	}

	resp, err := s.ListPoolVoluntaryExits(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, 2, len(resp.Data))
	assert.DeepEqual(t, migration.V1Alpha1ExitToV1(exit1), resp.Data[0])
	assert.DeepEqual(t, migration.V1Alpha1ExitToV1(exit2), resp.Data[1])
}

func TestSubmitAttesterSlashing_Ok(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	_, keys, err := util.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	validator := &ethpbv1alpha1.Validator{
		PublicKey: keys[0].PublicKey().Marshal(),
	}
	bs, err := util.NewBeaconState(func(state *ethpbv1alpha1.BeaconState) error {
		state.Validators = []*ethpbv1alpha1.Validator{validator}
		return nil
	})
	require.NoError(t, err)

	slashing := &ethpbv1.AttesterSlashing{
		Attestation_1: &ethpbv1.IndexedAttestation{
			AttestingIndices: []uint64{0},
			Data: &ethpbv1.AttestationData{
				Slot:            1,
				Index:           1,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot1"), 32),
				Source: &ethpbv1.Checkpoint{
					Epoch: 1,
					Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
				},
				Target: &ethpbv1.Checkpoint{
					Epoch: 10,
					Root:  bytesutil.PadTo([]byte("targetroot1"), 32),
				},
			},
			Signature: make([]byte, 96),
		},
		Attestation_2: &ethpbv1.IndexedAttestation{
			AttestingIndices: []uint64{0},
			Data: &ethpbv1.AttestationData{
				Slot:            1,
				Index:           1,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot2"), 32),
				Source: &ethpbv1.Checkpoint{
					Epoch: 1,
					Root:  bytesutil.PadTo([]byte("sourceroot2"), 32),
				},
				Target: &ethpbv1.Checkpoint{
					Epoch: 10,
					Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
				},
			},
			Signature: make([]byte, 96),
		},
	}

	for _, att := range []*ethpbv1.IndexedAttestation{slashing.Attestation_1, slashing.Attestation_2} {
		sb, err := signing.ComputeDomainAndSign(bs, att.Data.Target.Epoch, att.Data, params.BeaconConfig().DomainBeaconAttester, keys[0])
		require.NoError(t, err)
		sig, err := bls.SignatureFromBytes(sb)
		require.NoError(t, err)
		att.Signature = sig.Marshal()
	}

	broadcaster := &p2pMock.MockBroadcaster{}
	s := &Server{
		ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
		SlashingsPool:    &slashingsmock.PoolMock{},
		Broadcaster:      broadcaster,
	}

	_, err = s.SubmitAttesterSlashing(ctx, slashing)
	require.NoError(t, err)
	pendingSlashings := s.SlashingsPool.PendingAttesterSlashings(ctx, bs, true)
	require.Equal(t, 1, len(pendingSlashings))
	assert.DeepEqual(t, migration.V1AttSlashingToV1Alpha1(slashing), pendingSlashings[0])
	assert.Equal(t, true, broadcaster.BroadcastCalled)
	require.Equal(t, 1, len(broadcaster.BroadcastMessages))
	_, ok := broadcaster.BroadcastMessages[0].(*ethpbv1alpha1.AttesterSlashing)
	assert.Equal(t, true, ok)
}

func TestSubmitAttesterSlashing_AcrossFork(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.AltairForkEpoch = 1
	params.OverrideBeaconConfig(config)

	bs, keys := util.DeterministicGenesisState(t, 1)

	slashing := &ethpbv1.AttesterSlashing{
		Attestation_1: &ethpbv1.IndexedAttestation{
			AttestingIndices: []uint64{0},
			Data: &ethpbv1.AttestationData{
				Slot:            params.BeaconConfig().SlotsPerEpoch,
				Index:           1,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot1"), 32),
				Source: &ethpbv1.Checkpoint{
					Epoch: 1,
					Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
				},
				Target: &ethpbv1.Checkpoint{
					Epoch: 10,
					Root:  bytesutil.PadTo([]byte("targetroot1"), 32),
				},
			},
			Signature: make([]byte, 96),
		},
		Attestation_2: &ethpbv1.IndexedAttestation{
			AttestingIndices: []uint64{0},
			Data: &ethpbv1.AttestationData{
				Slot:            params.BeaconConfig().SlotsPerEpoch,
				Index:           1,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot2"), 32),
				Source: &ethpbv1.Checkpoint{
					Epoch: 1,
					Root:  bytesutil.PadTo([]byte("sourceroot2"), 32),
				},
				Target: &ethpbv1.Checkpoint{
					Epoch: 10,
					Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
				},
			},
			Signature: make([]byte, 96),
		},
	}

	newBs := bs.Copy()
	newBs, err := transition.ProcessSlots(ctx, newBs, params.BeaconConfig().SlotsPerEpoch)
	require.NoError(t, err)

	for _, att := range []*ethpbv1.IndexedAttestation{slashing.Attestation_1, slashing.Attestation_2} {
		sb, err := signing.ComputeDomainAndSign(newBs, att.Data.Target.Epoch, att.Data, params.BeaconConfig().DomainBeaconAttester, keys[0])
		require.NoError(t, err)
		sig, err := bls.SignatureFromBytes(sb)
		require.NoError(t, err)
		att.Signature = sig.Marshal()
	}

	broadcaster := &p2pMock.MockBroadcaster{}
	s := &Server{
		ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
		SlashingsPool:    &slashingsmock.PoolMock{},
		Broadcaster:      broadcaster,
	}

	_, err = s.SubmitAttesterSlashing(ctx, slashing)
	require.NoError(t, err)
	pendingSlashings := s.SlashingsPool.PendingAttesterSlashings(ctx, bs, true)
	require.Equal(t, 1, len(pendingSlashings))
	assert.DeepEqual(t, migration.V1AttSlashingToV1Alpha1(slashing), pendingSlashings[0])
	assert.Equal(t, true, broadcaster.BroadcastCalled)
	require.Equal(t, 1, len(broadcaster.BroadcastMessages))
	_, ok := broadcaster.BroadcastMessages[0].(*ethpbv1alpha1.AttesterSlashing)
	assert.Equal(t, true, ok)
}

func TestSubmitAttesterSlashing_InvalidSlashing(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	bs, err := util.NewBeaconState()
	require.NoError(t, err)

	attestation := &ethpbv1.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Data: &ethpbv1.AttestationData{
			Slot:            1,
			Index:           1,
			BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot1"), 32),
			Source: &ethpbv1.Checkpoint{
				Epoch: 1,
				Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
			},
			Target: &ethpbv1.Checkpoint{
				Epoch: 10,
				Root:  bytesutil.PadTo([]byte("targetroot1"), 32),
			},
		},
		Signature: make([]byte, 96),
	}

	slashing := &ethpbv1.AttesterSlashing{
		Attestation_1: attestation,
		Attestation_2: attestation,
	}

	broadcaster := &p2pMock.MockBroadcaster{}
	s := &Server{
		ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
		SlashingsPool:    &slashingsmock.PoolMock{},
		Broadcaster:      broadcaster,
	}

	_, err = s.SubmitAttesterSlashing(ctx, slashing)
	require.ErrorContains(t, "Invalid attester slashing", err)
	assert.Equal(t, false, broadcaster.BroadcastCalled)
}

func TestSubmitProposerSlashing_Ok(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	_, keys, err := util.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	validator := &ethpbv1alpha1.Validator{
		PublicKey:         keys[0].PublicKey().Marshal(),
		WithdrawableEpoch: primitives.Epoch(1),
	}
	bs, err := util.NewBeaconState(func(state *ethpbv1alpha1.BeaconState) error {
		state.Validators = []*ethpbv1alpha1.Validator{validator}
		return nil
	})
	require.NoError(t, err)

	slashing := &ethpbv1.ProposerSlashing{
		SignedHeader_1: &ethpbv1.SignedBeaconBlockHeader{
			Message: &ethpbv1.BeaconBlockHeader{
				Slot:          1,
				ProposerIndex: 0,
				ParentRoot:    bytesutil.PadTo([]byte("parentroot1"), 32),
				StateRoot:     bytesutil.PadTo([]byte("stateroot1"), 32),
				BodyRoot:      bytesutil.PadTo([]byte("bodyroot1"), 32),
			},
			Signature: make([]byte, 96),
		},
		SignedHeader_2: &ethpbv1.SignedBeaconBlockHeader{
			Message: &ethpbv1.BeaconBlockHeader{
				Slot:          1,
				ProposerIndex: 0,
				ParentRoot:    bytesutil.PadTo([]byte("parentroot2"), 32),
				StateRoot:     bytesutil.PadTo([]byte("stateroot2"), 32),
				BodyRoot:      bytesutil.PadTo([]byte("bodyroot2"), 32),
			},
			Signature: make([]byte, 96),
		},
	}

	for _, h := range []*ethpbv1.SignedBeaconBlockHeader{slashing.SignedHeader_1, slashing.SignedHeader_2} {
		sb, err := signing.ComputeDomainAndSign(
			bs,
			slots.ToEpoch(h.Message.Slot),
			h.Message,
			params.BeaconConfig().DomainBeaconProposer,
			keys[0],
		)
		require.NoError(t, err)
		sig, err := bls.SignatureFromBytes(sb)
		require.NoError(t, err)
		h.Signature = sig.Marshal()
	}

	broadcaster := &p2pMock.MockBroadcaster{}
	s := &Server{
		ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
		SlashingsPool:    &slashingsmock.PoolMock{},
		Broadcaster:      broadcaster,
	}

	_, err = s.SubmitProposerSlashing(ctx, slashing)
	require.NoError(t, err)
	pendingSlashings := s.SlashingsPool.PendingProposerSlashings(ctx, bs, true)
	require.Equal(t, 1, len(pendingSlashings))
	assert.DeepEqual(t, migration.V1ProposerSlashingToV1Alpha1(slashing), pendingSlashings[0])
	assert.Equal(t, true, broadcaster.BroadcastCalled)
	require.Equal(t, 1, len(broadcaster.BroadcastMessages))
	_, ok := broadcaster.BroadcastMessages[0].(*ethpbv1alpha1.ProposerSlashing)
	assert.Equal(t, true, ok)
}

func TestSubmitProposerSlashing_AcrossFork(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.AltairForkEpoch = 1
	params.OverrideBeaconConfig(config)

	bs, keys := util.DeterministicGenesisState(t, 1)

	slashing := &ethpbv1.ProposerSlashing{
		SignedHeader_1: &ethpbv1.SignedBeaconBlockHeader{
			Message: &ethpbv1.BeaconBlockHeader{
				Slot:          params.BeaconConfig().SlotsPerEpoch,
				ProposerIndex: 0,
				ParentRoot:    bytesutil.PadTo([]byte("parentroot1"), 32),
				StateRoot:     bytesutil.PadTo([]byte("stateroot1"), 32),
				BodyRoot:      bytesutil.PadTo([]byte("bodyroot1"), 32),
			},
			Signature: make([]byte, 96),
		},
		SignedHeader_2: &ethpbv1.SignedBeaconBlockHeader{
			Message: &ethpbv1.BeaconBlockHeader{
				Slot:          params.BeaconConfig().SlotsPerEpoch,
				ProposerIndex: 0,
				ParentRoot:    bytesutil.PadTo([]byte("parentroot2"), 32),
				StateRoot:     bytesutil.PadTo([]byte("stateroot2"), 32),
				BodyRoot:      bytesutil.PadTo([]byte("bodyroot2"), 32),
			},
			Signature: make([]byte, 96),
		},
	}

	newBs := bs.Copy()
	newBs, err := transition.ProcessSlots(ctx, newBs, params.BeaconConfig().SlotsPerEpoch)
	require.NoError(t, err)

	for _, h := range []*ethpbv1.SignedBeaconBlockHeader{slashing.SignedHeader_1, slashing.SignedHeader_2} {
		sb, err := signing.ComputeDomainAndSign(
			newBs,
			slots.ToEpoch(h.Message.Slot),
			h.Message,
			params.BeaconConfig().DomainBeaconProposer,
			keys[0],
		)
		require.NoError(t, err)
		sig, err := bls.SignatureFromBytes(sb)
		require.NoError(t, err)
		h.Signature = sig.Marshal()
	}

	broadcaster := &p2pMock.MockBroadcaster{}
	s := &Server{
		ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
		SlashingsPool:    &slashingsmock.PoolMock{},
		Broadcaster:      broadcaster,
	}

	_, err = s.SubmitProposerSlashing(ctx, slashing)
	require.NoError(t, err)
	pendingSlashings := s.SlashingsPool.PendingProposerSlashings(ctx, bs, true)
	require.Equal(t, 1, len(pendingSlashings))
	assert.DeepEqual(t, migration.V1ProposerSlashingToV1Alpha1(slashing), pendingSlashings[0])
	assert.Equal(t, true, broadcaster.BroadcastCalled)
	require.Equal(t, 1, len(broadcaster.BroadcastMessages))
	_, ok := broadcaster.BroadcastMessages[0].(*ethpbv1alpha1.ProposerSlashing)
	assert.Equal(t, true, ok)
}

func TestSubmitProposerSlashing_InvalidSlashing(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	bs, err := util.NewBeaconState()
	require.NoError(t, err)

	header := &ethpbv1.SignedBeaconBlockHeader{
		Message: &ethpbv1.BeaconBlockHeader{
			Slot:          1,
			ProposerIndex: 0,
			ParentRoot:    bytesutil.PadTo([]byte("parentroot1"), 32),
			StateRoot:     bytesutil.PadTo([]byte("stateroot1"), 32),
			BodyRoot:      bytesutil.PadTo([]byte("bodyroot1"), 32),
		},
		Signature: make([]byte, 96),
	}

	slashing := &ethpbv1.ProposerSlashing{
		SignedHeader_1: header,
		SignedHeader_2: header,
	}

	broadcaster := &p2pMock.MockBroadcaster{}
	s := &Server{
		ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
		SlashingsPool:    &slashingsmock.PoolMock{},
		Broadcaster:      broadcaster,
	}

	_, err = s.SubmitProposerSlashing(ctx, slashing)
	require.ErrorContains(t, "Invalid proposer slashing", err)
	assert.Equal(t, false, broadcaster.BroadcastCalled)
}

func TestSubmitVoluntaryExit_Ok(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	_, keys, err := util.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	validator := &ethpbv1alpha1.Validator{
		ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		PublicKey: keys[0].PublicKey().Marshal(),
	}
	bs, err := util.NewBeaconState(func(state *ethpbv1alpha1.BeaconState) error {
		state.Validators = []*ethpbv1alpha1.Validator{validator}
		// Satisfy activity time required before exiting.
		state.Slot = params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().ShardCommitteePeriod))
		return nil
	})
	require.NoError(t, err)

	exit := &ethpbv1.SignedVoluntaryExit{
		Message: &ethpbv1.VoluntaryExit{
			Epoch:          0,
			ValidatorIndex: 0,
		},
		Signature: make([]byte, 96),
	}

	sb, err := signing.ComputeDomainAndSign(bs, exit.Message.Epoch, exit.Message, params.BeaconConfig().DomainVoluntaryExit, keys[0])
	require.NoError(t, err)
	sig, err := bls.SignatureFromBytes(sb)
	require.NoError(t, err)
	exit.Signature = sig.Marshal()

	broadcaster := &p2pMock.MockBroadcaster{}
	s := &Server{
		ChainInfoFetcher:   &blockchainmock.ChainService{State: bs},
		VoluntaryExitsPool: &mock.PoolMock{},
		Broadcaster:        broadcaster,
	}

	_, err = s.SubmitVoluntaryExit(ctx, exit)
	require.NoError(t, err)
	pendingExits, err := s.VoluntaryExitsPool.PendingExits()
	require.NoError(t, err)
	require.Equal(t, 1, len(pendingExits))
	assert.DeepEqual(t, migration.V1ExitToV1Alpha1(exit), pendingExits[0])
	assert.Equal(t, true, broadcaster.BroadcastCalled)
}

func TestSubmitVoluntaryExit_AcrossFork(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.AltairForkEpoch = params.BeaconConfig().ShardCommitteePeriod + 1
	params.OverrideBeaconConfig(config)

	bs, keys := util.DeterministicGenesisState(t, 1)
	// Satisfy activity time required before exiting.
	require.NoError(t, bs.SetSlot(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().ShardCommitteePeriod))))

	exit := &ethpbv1.SignedVoluntaryExit{
		Message: &ethpbv1.VoluntaryExit{
			Epoch:          params.BeaconConfig().ShardCommitteePeriod + 1,
			ValidatorIndex: 0,
		},
		Signature: make([]byte, 96),
	}

	newBs := bs.Copy()
	newBs, err := transition.ProcessSlots(ctx, newBs, params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().ShardCommitteePeriod)+1))
	require.NoError(t, err)

	sb, err := signing.ComputeDomainAndSign(newBs, exit.Message.Epoch, exit.Message, params.BeaconConfig().DomainVoluntaryExit, keys[0])
	require.NoError(t, err)
	sig, err := bls.SignatureFromBytes(sb)
	require.NoError(t, err)
	exit.Signature = sig.Marshal()

	broadcaster := &p2pMock.MockBroadcaster{}
	s := &Server{
		ChainInfoFetcher:   &blockchainmock.ChainService{State: bs},
		VoluntaryExitsPool: &mock.PoolMock{},
		Broadcaster:        broadcaster,
	}

	_, err = s.SubmitVoluntaryExit(ctx, exit)
	require.NoError(t, err)
}

func TestSubmitVoluntaryExit_InvalidValidatorIndex(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	_, keys, err := util.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	validator := &ethpbv1alpha1.Validator{
		ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		PublicKey: keys[0].PublicKey().Marshal(),
	}
	bs, err := util.NewBeaconState(func(state *ethpbv1alpha1.BeaconState) error {
		state.Validators = []*ethpbv1alpha1.Validator{validator}
		return nil
	})
	require.NoError(t, err)

	exit := &ethpbv1.SignedVoluntaryExit{
		Message: &ethpbv1.VoluntaryExit{
			Epoch:          0,
			ValidatorIndex: 99,
		},
		Signature: make([]byte, 96),
	}

	broadcaster := &p2pMock.MockBroadcaster{}
	s := &Server{
		ChainInfoFetcher:   &blockchainmock.ChainService{State: bs},
		VoluntaryExitsPool: &mock.PoolMock{},
		Broadcaster:        broadcaster,
	}

	_, err = s.SubmitVoluntaryExit(ctx, exit)
	require.ErrorContains(t, "Could not get exiting validator", err)
	assert.Equal(t, false, broadcaster.BroadcastCalled)
}

func TestSubmitVoluntaryExit_InvalidExit(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	_, keys, err := util.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	validator := &ethpbv1alpha1.Validator{
		ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		PublicKey: keys[0].PublicKey().Marshal(),
	}
	bs, err := util.NewBeaconState(func(state *ethpbv1alpha1.BeaconState) error {
		state.Validators = []*ethpbv1alpha1.Validator{validator}
		return nil
	})
	require.NoError(t, err)

	exit := &ethpbv1.SignedVoluntaryExit{
		Message: &ethpbv1.VoluntaryExit{
			Epoch:          0,
			ValidatorIndex: 0,
		},
		Signature: make([]byte, 96),
	}

	broadcaster := &p2pMock.MockBroadcaster{}
	s := &Server{
		ChainInfoFetcher:   &blockchainmock.ChainService{State: bs},
		VoluntaryExitsPool: &mock.PoolMock{},
		Broadcaster:        broadcaster,
	}

	_, err = s.SubmitVoluntaryExit(ctx, exit)
	require.ErrorContains(t, "Invalid voluntary exit", err)
	assert.Equal(t, false, broadcaster.BroadcastCalled)
}

func TestListBLSToExecutionChanges(t *testing.T) {
	change1 := &ethpbv1alpha1.SignedBLSToExecutionChange{
		Message: &ethpbv1alpha1.BLSToExecutionChange{
			ValidatorIndex:     1,
			FromBlsPubkey:      bytesutil.PadTo([]byte("pubkey1"), 48),
			ToExecutionAddress: bytesutil.PadTo([]byte("address1"), 20),
		},
		Signature: bytesutil.PadTo([]byte("signature1"), 96),
	}
	change2 := &ethpbv1alpha1.SignedBLSToExecutionChange{
		Message: &ethpbv1alpha1.BLSToExecutionChange{
			ValidatorIndex:     2,
			FromBlsPubkey:      bytesutil.PadTo([]byte("pubkey2"), 48),
			ToExecutionAddress: bytesutil.PadTo([]byte("address2"), 20),
		},
		Signature: bytesutil.PadTo([]byte("signature2"), 96),
	}

	s := &Server{
		BLSChangesPool: &blstoexecmock.PoolMock{Changes: []*ethpbv1alpha1.SignedBLSToExecutionChange{change1, change2}},
	}

	resp, err := s.ListBLSToExecutionChanges(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, 2, len(resp.Data))
	assert.DeepEqual(t, migration.V1Alpha1SignedBLSToExecChangeToV2(change1), resp.Data[0])
	assert.DeepEqual(t, migration.V1Alpha1SignedBLSToExecChangeToV2(change2), resp.Data[1])
}

func TestSubmitSignedBLSToExecutionChanges_Ok(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	params.SetupTestConfigCleanup(t)
	c := params.BeaconConfig().Copy()
	// Required for correct committee size calculation.
	c.CapellaForkEpoch = c.BellatrixForkEpoch.Add(2)
	params.OverrideBeaconConfig(c)

	spb := &ethpbv1alpha1.BeaconStateCapella{
		Fork: &ethpbv1alpha1.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			Epoch:           params.BeaconConfig().CapellaForkEpoch,
		},
	}
	numValidators := 10
	validators := make([]*ethpbv1alpha1.Validator, numValidators)
	blsChanges := make([]*ethpbv2.BLSToExecutionChange, numValidators)
	spb.Balances = make([]uint64, numValidators)
	privKeys := make([]common.SecretKey, numValidators)
	maxEffectiveBalance := params.BeaconConfig().MaxEffectiveBalance
	executionAddress := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13}

	for i := range validators {
		v := &ethpbv1alpha1.Validator{}
		v.EffectiveBalance = maxEffectiveBalance
		v.WithdrawableEpoch = params.BeaconConfig().FarFutureEpoch
		v.WithdrawalCredentials = make([]byte, 32)
		priv, err := bls.RandKey()
		require.NoError(t, err)
		privKeys[i] = priv
		pubkey := priv.PublicKey().Marshal()

		message := &ethpbv2.BLSToExecutionChange{
			ToExecutionAddress: executionAddress,
			ValidatorIndex:     primitives.ValidatorIndex(i),
			FromBlsPubkey:      pubkey,
		}

		hashFn := ssz.NewHasherFunc(hash.CustomSHA256Hasher())
		digest := hashFn.Hash(pubkey)
		digest[0] = params.BeaconConfig().BLSWithdrawalPrefixByte
		copy(v.WithdrawalCredentials, digest[:])
		validators[i] = v
		blsChanges[i] = message
	}
	spb.Validators = validators
	slot, err := slots.EpochStart(params.BeaconConfig().CapellaForkEpoch)
	require.NoError(t, err)
	spb.Slot = slot
	st, err := state_native.InitializeFromProtoCapella(spb)
	require.NoError(t, err)

	signedChanges := make([]*ethpbv2.SignedBLSToExecutionChange, numValidators)
	for i, message := range blsChanges {
		signature, err := signing.ComputeDomainAndSign(st, prysmtime.CurrentEpoch(st), message, params.BeaconConfig().DomainBLSToExecutionChange, privKeys[i])
		require.NoError(t, err)

		signed := &ethpbv2.SignedBLSToExecutionChange{
			Message:   message,
			Signature: signature,
		}
		signedChanges[i] = signed
	}

	broadcaster := &p2pMock.MockBroadcaster{}
	chainService := &blockchainmock.ChainService{State: st}
	s := &Server{
		HeadFetcher:       chainService,
		ChainInfoFetcher:  chainService,
		AttestationsPool:  attestations.NewPool(),
		Broadcaster:       broadcaster,
		OperationNotifier: &blockchainmock.MockOperationNotifier{},
		BLSChangesPool:    blstoexec.NewPool(),
	}

	_, err = s.SubmitSignedBLSToExecutionChanges(ctx, &ethpbv2.SubmitBLSToExecutionChangesRequest{
		Changes: signedChanges,
	})
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond) // Delay to let the routine start
	assert.Equal(t, true, broadcaster.BroadcastCalled)
	assert.Equal(t, numValidators, len(broadcaster.BroadcastMessages))

	poolChanges, err := s.BLSChangesPool.PendingBLSToExecChanges()
	require.Equal(t, len(poolChanges), len(signedChanges))
	require.NoError(t, err)
	for i, v1alphaChange := range poolChanges {
		v2Change := migration.V1Alpha1SignedBLSToExecChangeToV2(v1alphaChange)
		require.DeepEqual(t, v2Change, signedChanges[i])
	}
}

func TestSubmitSignedBLSToExecutionChanges_Bellatrix(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	params.SetupTestConfigCleanup(t)
	c := params.BeaconConfig().Copy()
	// Required for correct committee size calculation.
	c.CapellaForkEpoch = c.BellatrixForkEpoch.Add(2)
	params.OverrideBeaconConfig(c)

	spb := &ethpbv1alpha1.BeaconStateBellatrix{
		Fork: &ethpbv1alpha1.Fork{
			CurrentVersion:  params.BeaconConfig().BellatrixForkVersion,
			PreviousVersion: params.BeaconConfig().AltairForkVersion,
			Epoch:           params.BeaconConfig().BellatrixForkEpoch,
		},
	}
	numValidators := 10
	validators := make([]*ethpbv1alpha1.Validator, numValidators)
	blsChanges := make([]*ethpbv2.BLSToExecutionChange, numValidators)
	spb.Balances = make([]uint64, numValidators)
	privKeys := make([]common.SecretKey, numValidators)
	maxEffectiveBalance := params.BeaconConfig().MaxEffectiveBalance
	executionAddress := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13}

	for i := range validators {
		v := &ethpbv1alpha1.Validator{}
		v.EffectiveBalance = maxEffectiveBalance
		v.WithdrawableEpoch = params.BeaconConfig().FarFutureEpoch
		v.WithdrawalCredentials = make([]byte, 32)
		priv, err := bls.RandKey()
		require.NoError(t, err)
		privKeys[i] = priv
		pubkey := priv.PublicKey().Marshal()

		message := &ethpbv2.BLSToExecutionChange{
			ToExecutionAddress: executionAddress,
			ValidatorIndex:     primitives.ValidatorIndex(i),
			FromBlsPubkey:      pubkey,
		}

		hashFn := ssz.NewHasherFunc(hash.CustomSHA256Hasher())
		digest := hashFn.Hash(pubkey)
		digest[0] = params.BeaconConfig().BLSWithdrawalPrefixByte
		copy(v.WithdrawalCredentials, digest[:])
		validators[i] = v
		blsChanges[i] = message
	}
	spb.Validators = validators
	slot, err := slots.EpochStart(params.BeaconConfig().BellatrixForkEpoch)
	require.NoError(t, err)
	spb.Slot = slot
	st, err := state_native.InitializeFromProtoBellatrix(spb)
	require.NoError(t, err)

	spc := &ethpbv1alpha1.BeaconStateCapella{
		Fork: &ethpbv1alpha1.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			Epoch:           params.BeaconConfig().CapellaForkEpoch,
		},
	}
	slot, err = slots.EpochStart(params.BeaconConfig().CapellaForkEpoch)
	require.NoError(t, err)
	spc.Slot = slot

	stc, err := state_native.InitializeFromProtoCapella(spc)
	require.NoError(t, err)

	signedChanges := make([]*ethpbv2.SignedBLSToExecutionChange, numValidators)
	for i, message := range blsChanges {
		signature, err := signing.ComputeDomainAndSign(stc, prysmtime.CurrentEpoch(stc), message, params.BeaconConfig().DomainBLSToExecutionChange, privKeys[i])
		require.NoError(t, err)

		signed := &ethpbv2.SignedBLSToExecutionChange{
			Message:   message,
			Signature: signature,
		}
		signedChanges[i] = signed
	}

	broadcaster := &p2pMock.MockBroadcaster{}
	chainService := &blockchainmock.ChainService{State: st}
	s := &Server{
		HeadFetcher:       chainService,
		ChainInfoFetcher:  chainService,
		AttestationsPool:  attestations.NewPool(),
		Broadcaster:       broadcaster,
		OperationNotifier: &blockchainmock.MockOperationNotifier{},
		BLSChangesPool:    blstoexec.NewPool(),
	}

	_, err = s.SubmitSignedBLSToExecutionChanges(ctx, &ethpbv2.SubmitBLSToExecutionChangesRequest{
		Changes: signedChanges,
	})
	require.NoError(t, err)

	// Check that we didn't broadcast the messages but did in fact fill in
	// the pool
	assert.Equal(t, false, broadcaster.BroadcastCalled)

	poolChanges, err := s.BLSChangesPool.PendingBLSToExecChanges()
	require.Equal(t, len(poolChanges), len(signedChanges))
	require.NoError(t, err)
	for i, v1alphaChange := range poolChanges {
		v2Change := migration.V1Alpha1SignedBLSToExecChangeToV2(v1alphaChange)
		require.DeepEqual(t, v2Change, signedChanges[i])
	}
}

func TestSubmitSignedBLSToExecutionChanges_Failures(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	params.SetupTestConfigCleanup(t)
	c := params.BeaconConfig().Copy()
	// Required for correct committee size calculation.
	c.CapellaForkEpoch = c.BellatrixForkEpoch.Add(2)
	params.OverrideBeaconConfig(c)

	spb := &ethpbv1alpha1.BeaconStateCapella{
		Fork: &ethpbv1alpha1.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			Epoch:           params.BeaconConfig().CapellaForkEpoch,
		},
	}
	numValidators := 10
	validators := make([]*ethpbv1alpha1.Validator, numValidators)
	blsChanges := make([]*ethpbv2.BLSToExecutionChange, numValidators)
	spb.Balances = make([]uint64, numValidators)
	privKeys := make([]common.SecretKey, numValidators)
	maxEffectiveBalance := params.BeaconConfig().MaxEffectiveBalance
	executionAddress := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13}

	for i := range validators {
		v := &ethpbv1alpha1.Validator{}
		v.EffectiveBalance = maxEffectiveBalance
		v.WithdrawableEpoch = params.BeaconConfig().FarFutureEpoch
		v.WithdrawalCredentials = make([]byte, 32)
		priv, err := bls.RandKey()
		require.NoError(t, err)
		privKeys[i] = priv
		pubkey := priv.PublicKey().Marshal()

		message := &ethpbv2.BLSToExecutionChange{
			ToExecutionAddress: executionAddress,
			ValidatorIndex:     primitives.ValidatorIndex(i),
			FromBlsPubkey:      pubkey,
		}

		hashFn := ssz.NewHasherFunc(hash.CustomSHA256Hasher())
		digest := hashFn.Hash(pubkey)
		digest[0] = params.BeaconConfig().BLSWithdrawalPrefixByte
		copy(v.WithdrawalCredentials, digest[:])
		validators[i] = v
		blsChanges[i] = message
	}
	spb.Validators = validators
	slot, err := slots.EpochStart(params.BeaconConfig().CapellaForkEpoch)
	require.NoError(t, err)
	spb.Slot = slot
	st, err := state_native.InitializeFromProtoCapella(spb)
	require.NoError(t, err)

	signedChanges := make([]*ethpbv2.SignedBLSToExecutionChange, numValidators)
	for i, message := range blsChanges {
		signature, err := signing.ComputeDomainAndSign(st, prysmtime.CurrentEpoch(st), message, params.BeaconConfig().DomainBLSToExecutionChange, privKeys[i])
		require.NoError(t, err)

		signed := &ethpbv2.SignedBLSToExecutionChange{
			Message:   message,
			Signature: signature,
		}
		signedChanges[i] = signed
	}
	signedChanges[1].Signature[0] = 0x00

	broadcaster := &p2pMock.MockBroadcaster{}
	chainService := &blockchainmock.ChainService{State: st}
	s := &Server{
		HeadFetcher:       chainService,
		ChainInfoFetcher:  chainService,
		AttestationsPool:  attestations.NewPool(),
		Broadcaster:       broadcaster,
		OperationNotifier: &blockchainmock.MockOperationNotifier{},
		BLSChangesPool:    blstoexec.NewPool(),
	}

	_, err = s.SubmitSignedBLSToExecutionChanges(ctx, &ethpbv2.SubmitBLSToExecutionChangesRequest{
		Changes: signedChanges,
	})
	time.Sleep(10 * time.Millisecond) // Delay to allow the routine to start
	require.ErrorContains(t, "One or more BLSToExecutionChange failed validation", err)
	assert.Equal(t, true, broadcaster.BroadcastCalled)
	assert.Equal(t, numValidators, len(broadcaster.BroadcastMessages)+1)

	poolChanges, err := s.BLSChangesPool.PendingBLSToExecChanges()
	require.Equal(t, len(poolChanges)+1, len(signedChanges))
	require.NoError(t, err)

	v2Change := migration.V1Alpha1SignedBLSToExecChangeToV2(poolChanges[0])
	require.DeepEqual(t, v2Change, signedChanges[0])
	for i := 2; i < numValidators; i++ {
		v2Change := migration.V1Alpha1SignedBLSToExecChangeToV2(poolChanges[i-1])
		require.DeepEqual(t, v2Change, signedChanges[i])
	}
}

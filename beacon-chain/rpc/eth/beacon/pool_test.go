package beacon

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prysmaticlabs/go-bitfield"
	grpcutil "github.com/prysmaticlabs/prysm/v3/api/grpc"
	blockchainmock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/attestations"
	slashingsmock "github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/slashings/mock"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/voluntaryexits/mock"
	p2pMock "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	eth2types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v3/proto/migration"
	ethpbv1alpha1 "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestListPoolAttestations(t *testing.T) {
	bs, err := util.NewBeaconState()
	require.NoError(t, err)
	att1 := &ethpbv1alpha1.Attestation{
		AggregationBits: []byte{1, 10},
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
	}
	att2 := &ethpbv1alpha1.Attestation{
		AggregationBits: []byte{4, 40},
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
	}
	att3 := &ethpbv1alpha1.Attestation{
		AggregationBits: []byte{2, 20},
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
	}
	att4 := &ethpbv1alpha1.Attestation{
		AggregationBits: bitfield.NewBitlist(8),
		Data: &ethpbv1alpha1.AttestationData{
			Slot:            4,
			CommitteeIndex:  4,
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
	}
	att5 := &ethpbv1alpha1.Attestation{
		AggregationBits: bitfield.NewBitlist(8),
		Data: &ethpbv1alpha1.AttestationData{
			Slot:            2,
			CommitteeIndex:  4,
			BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot1"), 32),
			Source: &ethpbv1alpha1.Checkpoint{
				Epoch: 2,
				Root:  bytesutil.PadTo([]byte("sourceroot2"), 32),
			},
			Target: &ethpbv1alpha1.Checkpoint{
				Epoch: 20,
				Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
			},
		},
		Signature: bytesutil.PadTo([]byte("signature1"), 96),
	}
	att6 := &ethpbv1alpha1.Attestation{
		AggregationBits: bitfield.NewBitlist(8),
		Data: &ethpbv1alpha1.AttestationData{
			Slot:            2,
			CommitteeIndex:  4,
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
	}
	s := &Server{
		ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
		AttestationsPool: attestations.NewPool(),
	}
	require.NoError(t, s.AttestationsPool.SaveAggregatedAttestations([]*ethpbv1alpha1.Attestation{att1, att2, att3}))
	require.NoError(t, s.AttestationsPool.SaveUnaggregatedAttestations([]*ethpbv1alpha1.Attestation{att4, att5, att6}))

	t.Run("empty request", func(t *testing.T) {
		req := &ethpbv1.AttestationsPoolRequest{}
		resp, err := s.ListPoolAttestations(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, 6, len(resp.Data))
	})

	t.Run("slot request", func(t *testing.T) {
		slot := eth2types.Slot(2)
		req := &ethpbv1.AttestationsPoolRequest{
			Slot: &slot,
		}
		resp, err := s.ListPoolAttestations(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, 3, len(resp.Data))
		for _, datum := range resp.Data {
			assert.DeepEqual(t, datum.Data.Slot, slot)
		}
	})

	t.Run("index request", func(t *testing.T) {
		index := eth2types.CommitteeIndex(4)
		req := &ethpbv1.AttestationsPoolRequest{
			CommitteeIndex: &index,
		}
		resp, err := s.ListPoolAttestations(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, 4, len(resp.Data))
		for _, datum := range resp.Data {
			assert.DeepEqual(t, datum.Data.Index, index)
		}
	})

	t.Run("both slot + index request", func(t *testing.T) {
		slot := eth2types.Slot(2)
		index := eth2types.CommitteeIndex(4)
		req := &ethpbv1.AttestationsPoolRequest{
			Slot:           &slot,
			CommitteeIndex: &index,
		}
		resp, err := s.ListPoolAttestations(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, 2, len(resp.Data))
		for _, datum := range resp.Data {
			assert.DeepEqual(t, datum.Data.Index, index)
			assert.DeepEqual(t, datum.Data.Slot, slot)
		}
	})
}

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
}

func TestSubmitAttesterSlashing_AcrossFork(t *testing.T) {
	ctx := context.Background()

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
}

func TestSubmitAttesterSlashing_InvalidSlashing(t *testing.T) {
	ctx := context.Background()
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

	_, keys, err := util.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	validator := &ethpbv1alpha1.Validator{
		PublicKey:         keys[0].PublicKey().Marshal(),
		WithdrawableEpoch: eth2types.Epoch(1),
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
}

func TestSubmitProposerSlashing_AcrossFork(t *testing.T) {
	ctx := context.Background()

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
}

func TestSubmitProposerSlashing_InvalidSlashing(t *testing.T) {
	ctx := context.Background()
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
	pendingExits := s.VoluntaryExitsPool.PendingExits(bs, bs.Slot(), true)
	require.Equal(t, 1, len(pendingExits))
	assert.DeepEqual(t, migration.V1ExitToV1Alpha1(exit), pendingExits[0])
	assert.Equal(t, true, broadcaster.BroadcastCalled)
}

func TestSubmitVoluntaryExit_AcrossFork(t *testing.T) {
	ctx := context.Background()

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

func TestServer_SubmitAttestations_Ok(t *testing.T) {
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	c := params.BeaconConfig().Copy()
	// Required for correct committee size calculation.
	c.SlotsPerEpoch = 1
	params.OverrideBeaconConfig(c)

	_, keys, err := util.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	validators := []*ethpbv1alpha1.Validator{
		{
			PublicKey: keys[0].PublicKey().Marshal(),
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	bs, err := util.NewBeaconState(func(state *ethpbv1alpha1.BeaconState) error {
		state.Validators = validators
		state.Slot = 1
		state.PreviousJustifiedCheckpoint = &ethpbv1alpha1.Checkpoint{
			Epoch: 0,
			Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
		}
		return nil
	})
	require.NoError(t, err)
	b := bitfield.NewBitlist(1)
	b.SetBitAt(0, true)

	sourceCheckpoint := &ethpbv1.Checkpoint{
		Epoch: 0,
		Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
	}
	att1 := &ethpbv1.Attestation{
		AggregationBits: b,
		Data: &ethpbv1.AttestationData{
			Slot:            0,
			Index:           0,
			BeaconBlockRoot: bytesutil.PadTo([]byte("beaconblockroot1"), 32),
			Source:          sourceCheckpoint,
			Target: &ethpbv1.Checkpoint{
				Epoch: 0,
				Root:  bytesutil.PadTo([]byte("targetroot1"), 32),
			},
		},
		Signature: make([]byte, 96),
	}
	att2 := &ethpbv1.Attestation{
		AggregationBits: b,
		Data: &ethpbv1.AttestationData{
			Slot:            0,
			Index:           0,
			BeaconBlockRoot: bytesutil.PadTo([]byte("beaconblockroot2"), 32),
			Source:          sourceCheckpoint,
			Target: &ethpbv1.Checkpoint{
				Epoch: 0,
				Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
			},
		},
		Signature: make([]byte, 96),
	}

	for _, att := range []*ethpbv1.Attestation{att1, att2} {
		sb, err := signing.ComputeDomainAndSign(
			bs,
			slots.ToEpoch(att.Data.Slot),
			att.Data,
			params.BeaconConfig().DomainBeaconAttester,
			keys[0],
		)
		require.NoError(t, err)
		sig, err := bls.SignatureFromBytes(sb)
		require.NoError(t, err)
		att.Signature = sig.Marshal()
	}

	broadcaster := &p2pMock.MockBroadcaster{}
	chainService := &blockchainmock.ChainService{State: bs}
	s := &Server{
		HeadFetcher:       chainService,
		ChainInfoFetcher:  chainService,
		AttestationsPool:  attestations.NewPool(),
		Broadcaster:       broadcaster,
		OperationNotifier: &blockchainmock.MockOperationNotifier{},
	}

	_, err = s.SubmitAttestations(ctx, &ethpbv1.SubmitAttestationsRequest{
		Data: []*ethpbv1.Attestation{att1, att2},
	})
	require.NoError(t, err)
	assert.Equal(t, true, broadcaster.BroadcastCalled)
	assert.Equal(t, 2, len(broadcaster.BroadcastAttestations))
	expectedAtt1, err := att1.HashTreeRoot()
	require.NoError(t, err)
	expectedAtt2, err := att2.HashTreeRoot()
	require.NoError(t, err)
	actualAtt1, err := broadcaster.BroadcastAttestations[0].HashTreeRoot()
	require.NoError(t, err)
	actualAtt2, err := broadcaster.BroadcastAttestations[1].HashTreeRoot()
	require.NoError(t, err)
	for _, r := range [][32]byte{actualAtt1, actualAtt2} {
		assert.Equal(t, true, reflect.DeepEqual(expectedAtt1, r) || reflect.DeepEqual(expectedAtt2, r))
	}
}

func TestServer_SubmitAttestations_ValidAttestationSubmitted(t *testing.T) {
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})

	params.SetupTestConfigCleanup(t)
	c := params.BeaconConfig().Copy()
	// Required for correct committee size calculation.
	c.SlotsPerEpoch = 1
	params.OverrideBeaconConfig(c)

	_, keys, err := util.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	validators := []*ethpbv1alpha1.Validator{
		{
			PublicKey: keys[0].PublicKey().Marshal(),
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	bs, err := util.NewBeaconState(func(state *ethpbv1alpha1.BeaconState) error {
		state.Validators = validators
		state.Slot = 1
		state.PreviousJustifiedCheckpoint = &ethpbv1alpha1.Checkpoint{
			Epoch: 0,
			Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
		}
		return nil
	})

	require.NoError(t, err)

	sourceCheckpoint := &ethpbv1.Checkpoint{
		Epoch: 0,
		Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
	}
	b := bitfield.NewBitlist(1)
	b.SetBitAt(0, true)
	attValid := &ethpbv1.Attestation{
		AggregationBits: b,
		Data: &ethpbv1.AttestationData{
			Slot:            0,
			Index:           0,
			BeaconBlockRoot: bytesutil.PadTo([]byte("beaconblockroot1"), 32),
			Source:          sourceCheckpoint,
			Target: &ethpbv1.Checkpoint{
				Epoch: 0,
				Root:  bytesutil.PadTo([]byte("targetroot1"), 32),
			},
		},
		Signature: make([]byte, 96),
	}
	attInvalidSignature := &ethpbv1.Attestation{
		AggregationBits: b,
		Data: &ethpbv1.AttestationData{
			Slot:            0,
			Index:           0,
			BeaconBlockRoot: bytesutil.PadTo([]byte("beaconblockroot2"), 32),
			Source:          sourceCheckpoint,
			Target: &ethpbv1.Checkpoint{
				Epoch: 0,
				Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
			},
		},
		Signature: make([]byte, 96),
	}

	// Don't sign attInvalidSignature.
	sb, err := signing.ComputeDomainAndSign(
		bs,
		slots.ToEpoch(attValid.Data.Slot),
		attValid.Data,
		params.BeaconConfig().DomainBeaconAttester,
		keys[0],
	)
	require.NoError(t, err)
	sig, err := bls.SignatureFromBytes(sb)
	require.NoError(t, err)
	attValid.Signature = sig.Marshal()

	broadcaster := &p2pMock.MockBroadcaster{}
	chainService := &blockchainmock.ChainService{State: bs}
	s := &Server{
		HeadFetcher:       chainService,
		ChainInfoFetcher:  chainService,
		AttestationsPool:  attestations.NewPool(),
		Broadcaster:       broadcaster,
		OperationNotifier: &blockchainmock.MockOperationNotifier{},
	}

	_, err = s.SubmitAttestations(ctx, &ethpbv1.SubmitAttestationsRequest{
		Data: []*ethpbv1.Attestation{attValid, attInvalidSignature},
	})
	require.ErrorContains(t, "One or more attestations failed validation", err)
	expectedAtt, err := attValid.HashTreeRoot()
	require.NoError(t, err)
	assert.Equal(t, true, broadcaster.BroadcastCalled)
	require.Equal(t, 1, len(broadcaster.BroadcastAttestations))
	broadcastRoot, err := broadcaster.BroadcastAttestations[0].HashTreeRoot()
	require.NoError(t, err)
	require.DeepEqual(t, expectedAtt, broadcastRoot)
}

func TestServer_SubmitAttestations_InvalidAttestationGRPCHeader(t *testing.T) {
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})

	params.SetupTestConfigCleanup(t)
	c := params.BeaconConfig().Copy()
	// Required for correct committee size calculation.
	c.SlotsPerEpoch = 1
	params.OverrideBeaconConfig(c)

	_, keys, err := util.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	validators := []*ethpbv1alpha1.Validator{
		{
			PublicKey: keys[0].PublicKey().Marshal(),
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	bs, err := util.NewBeaconState(func(state *ethpbv1alpha1.BeaconState) error {
		state.Validators = validators
		state.Slot = 1
		state.PreviousJustifiedCheckpoint = &ethpbv1alpha1.Checkpoint{
			Epoch: 0,
			Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
		}
		return nil
	})

	require.NoError(t, err)

	b := bitfield.NewBitlist(1)
	b.SetBitAt(0, true)
	att := &ethpbv1.Attestation{
		AggregationBits: b,
		Data: &ethpbv1.AttestationData{
			Slot:            0,
			Index:           0,
			BeaconBlockRoot: bytesutil.PadTo([]byte("beaconblockroot2"), 32),
			Source: &ethpbv1.Checkpoint{
				Epoch: 0,
				Root:  bytesutil.PadTo([]byte("sourceroot2"), 32),
			},
			Target: &ethpbv1.Checkpoint{
				Epoch: 1,
				Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
			},
		},
		Signature: nil,
	}

	chain := &blockchainmock.ChainService{State: bs}
	broadcaster := &p2pMock.MockBroadcaster{}
	s := &Server{
		ChainInfoFetcher:  chain,
		AttestationsPool:  attestations.NewPool(),
		Broadcaster:       broadcaster,
		OperationNotifier: &blockchainmock.MockOperationNotifier{},
		HeadFetcher:       chain,
	}

	_, err = s.SubmitAttestations(ctx, &ethpbv1.SubmitAttestationsRequest{
		Data: []*ethpbv1.Attestation{att},
	})
	require.ErrorContains(t, "One or more attestations failed validation", err)
	sts, ok := grpc.ServerTransportStreamFromContext(ctx).(*runtime.ServerTransportStream)
	require.Equal(t, true, ok, "type assertion failed")
	md := sts.Header()
	v, ok := md[strings.ToLower(grpcutil.CustomErrorMetadataKey)]
	require.Equal(t, true, ok, "could not retrieve custom error metadata value")
	assert.DeepEqual(
		t,
		[]string{"{\"failures\":[{\"index\":0,\"message\":\"Incorrect attestation signature: signature must be 96 bytes\"}]}"},
		v,
	)
}

package beaconv1

import (
	"context"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestServer_ListPoolAttesterSlashings(t *testing.T) {
	ctx := context.Background()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 64)
	count := uint64(10)
	slashingsInPool := make([]*ethpb_alpha.AttesterSlashing, count)
	v1Slashings := make([]*ethpb.AttesterSlashing, count)
	for i := 0; i < len(slashingsInPool); i++ {
		sl, err := testutil.GenerateAttesterSlashingForValidator(beaconState, privKeys[i], uint64(i))
		require.NoError(t, err)
		slashingsInPool[i] = sl
		marshaledSlashing, err := sl.Marshal()
		require.NoError(t, err)
		v1Slashing := &ethpb.AttesterSlashing{}
		require.NoError(t, proto.Unmarshal(marshaledSlashing, v1Slashing))
		v1Slashings[i] = v1Slashing
	}
	tests := []struct {
		name    string
		pending []*ethpb_alpha.AttesterSlashing
		want    []*ethpb.AttesterSlashing
	}{
		{
			name:    "Empty list",
			pending: []*ethpb_alpha.AttesterSlashing{},
			want:    []*ethpb.AttesterSlashing{},
		},
		{
			name:    "One",
			pending: slashingsInPool[0:1],
			want:    v1Slashings[0:1],
		},
		{
			name:    "All",
			pending: slashingsInPool,
			want:    v1Slashings,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &slashings.Pool{}
			for _, slashing := range tt.pending {
				require.NoError(t, pool.InsertAttesterSlashing(ctx, beaconState, slashing))
			}
			p := &Server{
				HeadFetcher:   &mock.ChainService{State: beaconState},
				SlashingsPool: pool,
			}
			attSlashings, err := p.ListPoolAttesterSlashings(ctx, &ptypes.Empty{})
			require.NoError(t, err)
			assert.DeepEqual(t, tt.want, attSlashings.Data)
		})
	}
}

func TestServer_ListPoolProposerSlashings(t *testing.T) {
	ctx := context.Background()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 64)
	count := uint64(10)
	slashingsInPool := make([]*ethpb_alpha.ProposerSlashing, count)
	v1Slashings := make([]*ethpb.ProposerSlashing, count)
	for i := 0; i < len(slashingsInPool); i++ {
		sl, err := testutil.GenerateProposerSlashingForValidator(beaconState, privKeys[i], uint64(i))
		require.NoError(t, err)
		slashingsInPool[i] = sl
		marshaledSlashing, err := sl.Marshal()
		require.NoError(t, err)
		v1Slashing := &ethpb.ProposerSlashing{}
		require.NoError(t, proto.Unmarshal(marshaledSlashing, v1Slashing))
		v1Slashings[i] = v1Slashing
	}
	tests := []struct {
		name    string
		pending []*ethpb_alpha.ProposerSlashing
		want    []*ethpb.ProposerSlashing
	}{
		{
			name:    "Empty list",
			pending: []*ethpb_alpha.ProposerSlashing{},
			want:    []*ethpb.ProposerSlashing{},
		},
		{
			name:    "One",
			pending: slashingsInPool[0:1],
			want:    v1Slashings[0:1],
		},
		{
			name:    "All",
			pending: slashingsInPool,
			want:    v1Slashings,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &slashings.Pool{}
			for _, slashing := range tt.pending {
				require.NoError(t, pool.InsertProposerSlashing(ctx, beaconState, slashing))
			}
			p := &Server{
				HeadFetcher:   &mock.ChainService{State: beaconState},
				SlashingsPool: pool,
			}
			attSlashings, err := p.ListPoolProposerSlashings(ctx, &ptypes.Empty{})
			require.NoError(t, err)
			assert.DeepEqual(t, tt.want, attSlashings.Data)
		})
	}
}

func TestServer_ListPoolVoluntaryExits(t *testing.T) {
	ctx := context.Background()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 64)
	count := uint64(10)
	exitsInPool := make([]*ethpb_alpha.SignedVoluntaryExit, count)
	v1Exits := make([]*ethpb.SignedVoluntaryExit, count)
	for i := 0; i < len(exitsInPool); i++ {
		exit, err := testutil.GenerateVoluntaryExit(beaconState, privKeys[i], uint64(i))
		require.NoError(t, err)
		exitsInPool[i] = exit
		marshaledExit, err := exit.Marshal()
		require.NoError(t, err)
		v1Exit := &ethpb.SignedVoluntaryExit{}
		require.NoError(t, proto.Unmarshal(marshaledExit, v1Exit))
		v1Exits[i] = v1Exit
	}
	tests := []struct {
		name    string
		pending []*ethpb_alpha.SignedVoluntaryExit
		want    []*ethpb.SignedVoluntaryExit
	}{
		{
			name:    "Empty list",
			pending: []*ethpb_alpha.SignedVoluntaryExit{},
			want:    []*ethpb.SignedVoluntaryExit{},
		},
		{
			name:    "One",
			pending: exitsInPool[0:1],
			want:    v1Exits[0:1],
		},
		{
			name:    "All",
			pending: exitsInPool,
			want:    v1Exits,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &voluntaryexits.Pool{}
			for _, slashing := range tt.pending {
				pool.InsertVoluntaryExit(ctx, beaconState, slashing)
			}
			p := &Server{
				HeadFetcher:        &mock.ChainService{State: beaconState},
				VoluntaryExitsPool: pool,
			}
			exits, err := p.ListPoolVoluntaryExits(ctx, &ptypes.Empty{})
			require.NoError(t, err)
			assert.DeepEqual(t, tt.want, exits.Data)
		})
	}
}

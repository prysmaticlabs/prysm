package beaconv1

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/proto/migration"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestListPoolAttesterSlashings(t *testing.T) {
	state, err := testutil.NewBeaconState()
	require.NoError(t, err)
	slashing1 := &eth.AttesterSlashing{
		Attestation_1: &eth.IndexedAttestation{
			AttestingIndices: []uint64{1, 10},
			Data: &eth.AttestationData{
				Slot:            1,
				CommitteeIndex:  1,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot1"), 32),
				Source: &eth.Checkpoint{
					Epoch: 1,
					Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
				},
				Target: &eth.Checkpoint{
					Epoch: 10,
					Root:  bytesutil.PadTo([]byte("targetroot1"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature1"), 96),
		},
		Attestation_2: &eth.IndexedAttestation{
			AttestingIndices: []uint64{2, 20},
			Data: &eth.AttestationData{
				Slot:            2,
				CommitteeIndex:  2,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot2"), 32),
				Source: &eth.Checkpoint{
					Epoch: 2,
					Root:  bytesutil.PadTo([]byte("sourceroot2"), 32),
				},
				Target: &eth.Checkpoint{
					Epoch: 20,
					Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature2"), 96),
		},
	}
	slashing2 := &eth.AttesterSlashing{
		Attestation_1: &eth.IndexedAttestation{
			AttestingIndices: []uint64{3, 30},
			Data: &eth.AttestationData{
				Slot:            3,
				CommitteeIndex:  3,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot3"), 32),
				Source: &eth.Checkpoint{
					Epoch: 3,
					Root:  bytesutil.PadTo([]byte("sourceroot3"), 32),
				},
				Target: &eth.Checkpoint{
					Epoch: 30,
					Root:  bytesutil.PadTo([]byte("targetroot3"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature3"), 96),
		},
		Attestation_2: &eth.IndexedAttestation{
			AttestingIndices: []uint64{4, 40},
			Data: &eth.AttestationData{
				Slot:            4,
				CommitteeIndex:  4,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot4"), 32),
				Source: &eth.Checkpoint{
					Epoch: 4,
					Root:  bytesutil.PadTo([]byte("sourceroot4"), 32),
				},
				Target: &eth.Checkpoint{
					Epoch: 40,
					Root:  bytesutil.PadTo([]byte("targetroot4"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature4"), 96),
		},
	}

	s := &Server{
		ChainInfoFetcher: &chainMock.ChainService{State: state},
		SlashingsPool:    &slashings.PoolMock{PendingAttSlashings: []*eth.AttesterSlashing{slashing1, slashing2}},
	}

	resp, err := s.ListPoolAttesterSlashings(context.Background(), &types.Empty{})
	require.NoError(t, err)
	require.Equal(t, 2, len(resp.Data))
	assert.DeepEqual(t, migration.V1Alpha1AttSlashingToV1(slashing1), resp.Data[0])
	assert.DeepEqual(t, migration.V1Alpha1AttSlashingToV1(slashing2), resp.Data[1])
}

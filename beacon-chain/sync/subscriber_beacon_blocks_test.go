package sync

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestDeleteAttsInPool(t *testing.T) {
	r := &Service{
		attPool: attestations.NewPool(),
	}
	data := &ethpb.AttestationData{
		BeaconBlockRoot: make([]byte, 32),
		Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
	}
	att1 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1101}, Data: data, Signature: make([]byte, 96)}
	att2 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1110}, Data: data, Signature: make([]byte, 96)}
	att3 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1011}, Data: data, Signature: make([]byte, 96)}
	att4 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1001}, Data: data, Signature: make([]byte, 96)}
	require.NoError(t, r.attPool.SaveAggregatedAttestation(att1))
	require.NoError(t, r.attPool.SaveAggregatedAttestation(att2))
	require.NoError(t, r.attPool.SaveAggregatedAttestation(att3))
	require.NoError(t, r.attPool.SaveUnaggregatedAttestation(att4))

	// Seen 1, 3 and 4 in block.
	require.NoError(t, r.deleteAttsInPool([]*ethpb.Attestation{att1, att3, att4}))

	// Only 2 should remain.
	assert.DeepEqual(t, []*ethpb.Attestation{att2}, r.attPool.AggregatedAttestations(), "Did not get wanted attestations")
}

func TestService_beaconBlockSubscriber(t *testing.T) {
	pooledAttestations := []*ethpb.Attestation{
		// Aggregated.
		{
			AggregationBits: bitfield.Bitlist{0b00011111},
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: make([]byte, 32),
				Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
				Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
			},
			Signature: make([]byte, 96),
		},
		// Unaggregated.
		{
			AggregationBits: bitfield.Bitlist{0b00010001},
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: make([]byte, 32),
				Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
				Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
			},
			Signature: make([]byte, 96),
		},
	}

	type args struct {
		msg proto.Message
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		check   func(*testing.T, *Service)
	}{
		{
			name: "invalid block does not remove attestations",
			args: args{
				msg: &ethpb.SignedBeaconBlock{
					Block: &ethpb.BeaconBlock{
						// An empty block will return an err in mocked chainService.ReceiveBlock.
						Body: &ethpb.BeaconBlockBody{Attestations: pooledAttestations},
					},
				},
			},
			wantErr: true,
			check: func(t *testing.T, s *Service) {
				if s.attPool.AggregatedAttestationCount() == 0 {
					t.Error("Expected at least 1 aggregated attestation in the pool")
				}
				if s.attPool.UnaggregatedAttestationCount() == 0 {
					t.Error("Expected at least 1 unaggregated attestation in the pool")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _ := dbtest.SetupDB(t)
			s := &Service{
				chain: &chainMock.ChainService{
					DB: db,
				},
				attPool: attestations.NewPool(),
			}
			assert.NoError(t, s.initCaches())
			// Set up attestation pool.
			for _, att := range pooledAttestations {
				if helpers.IsAggregated(att) {
					assert.NoError(t, s.attPool.SaveAggregatedAttestation(att))
				} else {
					assert.NoError(t, s.attPool.SaveUnaggregatedAttestation(att))
				}
			}
			// Perform method under test call.
			if err := s.beaconBlockSubscriber(context.Background(), tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("beaconBlockSubscriber(ctx, msg) error = %v, wantErr %v", err, tt.wantErr)
			}
			// Perform any test check.
			if tt.check != nil {
				tt.check(t, s)
			}
		})
	}
}

package sync

import (
	"context"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
)

func TestDeleteAttsInPool(t *testing.T) {
	r := &Service{
		attPool: attestations.NewPool(),
	}
	data := &ethpb.AttestationData{}
	att1 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1101}, Data: data}
	att2 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1110}, Data: data}
	att3 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1011}, Data: data}
	att4 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1001}, Data: data}
	if err := r.attPool.SaveAggregatedAttestation(att1); err != nil {
		t.Fatal(err)
	}
	if err := r.attPool.SaveAggregatedAttestation(att2); err != nil {
		t.Fatal(err)
	}
	if err := r.attPool.SaveAggregatedAttestation(att3); err != nil {
		t.Fatal(err)
	}
	if err := r.attPool.SaveUnaggregatedAttestation(att4); err != nil {
		t.Fatal(err)
	}

	// Seen 1, 3 and 4 in block.
	if err := r.deleteAttsInPool([]*ethpb.Attestation{att1, att3, att4}); err != nil {
		t.Fatal(err)
	}

	// Only 2 should remain.
	if !reflect.DeepEqual(r.attPool.AggregatedAttestations(), []*ethpb.Attestation{att2}) {
		t.Error("Did not get wanted attestation from pool")
	}
}

func TestService_beaconBlockSubscriber(t *testing.T) {
	pooledAttestations := []*ethpb.Attestation{
		// Aggregated.
		{
			AggregationBits: bitfield.Bitlist{0b00011111},
			Data:            &ethpb.AttestationData{},
		},
		// Unaggregated.
		{
			AggregationBits: bitfield.Bitlist{0b00010001},
			Data:            &ethpb.AttestationData{},
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
						// An empty block will return an err in mocked chainService.ReceiveBlockNoPubsub.
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
			if err := s.initCaches(); err != nil {
				t.Error(err)
			}
			// Set up attestation pool.
			for _, att := range pooledAttestations {
				if helpers.IsAggregated(att) {
					if err := s.attPool.SaveAggregatedAttestation(att); err != nil {
						t.Error(err)
					}
				} else {
					if err := s.attPool.SaveUnaggregatedAttestation(att); err != nil {
						t.Error(err)
					}
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

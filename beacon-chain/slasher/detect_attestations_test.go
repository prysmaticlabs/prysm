package slasher

import (
	"context"
	"reflect"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

func TestService_groupByValidatorChunkIndex(t *testing.T) {
	tests := []struct {
		name   string
		params *Parameters
		atts   []*ethpb.IndexedAttestation
		want   map[uint64][]*ethpb.IndexedAttestation
	}{
		{
			name:   "No attestations returns empty map",
			params: DefaultParams(),
			atts:   make([]*ethpb.IndexedAttestation, 0),
			want:   make(map[uint64][]*ethpb.IndexedAttestation),
		},
		{
			name: "Groups multiple attestations belonging to single validator chunk",
			params: &Parameters{
				validatorChunkSize: 2,
			},
			atts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0, 1},
				},
				{
					AttestingIndices: []uint64{0, 1},
				},
			},
			want: map[uint64][]*ethpb.IndexedAttestation{
				0: {
					{
						AttestingIndices: []uint64{0, 1},
					},
					{
						AttestingIndices: []uint64{0, 1},
					},
				},
			},
		},
		{
			name: "Groups single attestation belonging to multiple validator chunk",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				params: tt.params,
			}
			if got := s.groupByValidatorChunkIndex(tt.atts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("groupByValidatorChunkIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_processQueuedAttestations(t *testing.T) {
	type fields struct {
		params           *Parameters
		serviceCfg       *ServiceConfig
		indexedAttsChan  chan *ethpb.IndexedAttestation
		attestationQueue []*ethpb.IndexedAttestation
		ctx              context.Context
		cancel           context.CancelFunc
		genesisTime      time.Time
	}
	type args struct {
		ctx         context.Context
		epochTicker <-chan uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				params:           tt.fields.params,
				serviceCfg:       tt.fields.serviceCfg,
				indexedAttsChan:  tt.fields.indexedAttsChan,
				attestationQueue: tt.fields.attestationQueue,
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				genesisTime:      tt.fields.genesisTime,
			}
		})
	}
}

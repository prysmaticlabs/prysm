package slasher

import (
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	logTest "github.com/sirupsen/logrus/hooks/test"
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
			params: &Parameters{
				validatorChunkSize: 2,
			},
			atts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0, 2, 4},
				},
			},
			want: map[uint64][]*ethpb.IndexedAttestation{
				0: {
					{
						AttestingIndices: []uint64{0, 2, 4},
					},
				},
				1: {
					{
						AttestingIndices: []uint64{0, 2, 4},
					},
				},
				2: {
					{
						AttestingIndices: []uint64{0, 2, 4},
					},
				},
			},
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
	hook := logTest.NewGlobal()
	s := &Service{
		params: DefaultParams(),
		attestationQueue: []*ethpb.IndexedAttestation{
			{
				AttestingIndices: []uint64{0, 1},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 0,
					},
					Target: &ethpb.Checkpoint{
						Epoch: 1,
					},
				},
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	tickerChan := make(chan uint64)
	exitChan := make(chan struct{})
	go func() {
		s.processQueuedAttestations(ctx, tickerChan)
		exitChan <- struct{}{}
	}()

	// Send a value over the ticker.
	tickerChan <- 0
	cancel()
	<-exitChan
	assert.LogsContain(t, hook, "Epoch 0 reached, processing 1 queued")
}

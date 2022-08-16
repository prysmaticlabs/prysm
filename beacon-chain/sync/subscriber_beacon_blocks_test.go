package sync

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	chainMock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	dbtest "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/execution"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/attestations"
	lruwrpr "github.com/prysmaticlabs/prysm/v3/cache/lru"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"google.golang.org/protobuf/proto"
)

func TestDeleteAttsInPool(t *testing.T) {
	r := &Service{
		cfg: &config{attPool: attestations.NewPool()},
	}

	att1 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1101}})
	att2 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1110}})
	att3 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1011}})
	att4 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1001}})
	require.NoError(t, r.cfg.attPool.SaveAggregatedAttestation(att1))
	require.NoError(t, r.cfg.attPool.SaveAggregatedAttestation(att2))
	require.NoError(t, r.cfg.attPool.SaveAggregatedAttestation(att3))
	require.NoError(t, r.cfg.attPool.SaveUnaggregatedAttestation(att4))

	// Seen 1, 3 and 4 in block.
	require.NoError(t, r.deleteAttsInPool([]*ethpb.Attestation{att1, att3, att4}))

	// Only 2 should remain.
	assert.DeepEqual(t, []*ethpb.Attestation{att2}, r.cfg.attPool.AggregatedAttestations(), "Did not get wanted attestations")
}

func TestService_beaconBlockSubscriber(t *testing.T) {
	pooledAttestations := []*ethpb.Attestation{
		// Aggregated.
		util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00011111}}),
		// Unaggregated.
		util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00010001}}),
	}

	type args struct {
		msg proto.Message
	}
	tests := []struct {
		name      string
		args      args
		wantedErr string
		check     func(*testing.T, *Service)
	}{
		{
			name: "invalid block does not remove attestations",
			args: args{
				msg: func() *ethpb.SignedBeaconBlock {
					b := util.NewBeaconBlock()
					b.Block.Body.Attestations = pooledAttestations
					return b
				}(),
			},
			wantedErr: chainMock.ErrNilState.Error(),
			check: func(t *testing.T, s *Service) {
				if s.cfg.attPool.AggregatedAttestationCount() == 0 {
					t.Error("Expected at least 1 aggregated attestation in the pool")
				}
				if s.cfg.attPool.UnaggregatedAttestationCount() == 0 {
					t.Error("Expected at least 1 unaggregated attestation in the pool")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := dbtest.SetupDB(t)
			s := &Service{
				cfg: &config{
					chain: &chainMock.ChainService{
						DB:   db,
						Root: make([]byte, 32),
					},
					attPool: attestations.NewPool(),
				},
			}
			s.initCaches()
			// Set up attestation pool.
			for _, att := range pooledAttestations {
				if helpers.IsAggregated(att) {
					assert.NoError(t, s.cfg.attPool.SaveAggregatedAttestation(att))
				} else {
					assert.NoError(t, s.cfg.attPool.SaveUnaggregatedAttestation(att))
				}
			}
			// Perform method under test call.
			err := s.beaconBlockSubscriber(context.Background(), tt.args.msg)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.check != nil {
				tt.check(t, s)
			}
		})
	}
}

func TestService_BeaconBlockSubscribe_ExecutionEngineTimesOut(t *testing.T) {
	s := &Service{
		cfg: &config{
			chain: &chainMock.ChainService{
				ReceiveBlockMockErr: execution.ErrHTTPTimeout,
			},
		},
		seenBlockCache: lruwrpr.New(10),
		badBlockCache:  lruwrpr.New(10),
	}
	require.ErrorIs(t, execution.ErrHTTPTimeout, s.beaconBlockSubscriber(context.Background(), util.NewBeaconBlock()))
	require.Equal(t, 0, len(s.badBlockCache.Keys()))
	require.Equal(t, 1, len(s.seenBlockCache.Keys()))
}

func TestService_BeaconBlockSubscribe_UndefinedEeError(t *testing.T) {
	msg := "timeout"
	err := errors.WithMessage(blockchain.ErrUndefinedExecutionEngineError, msg)

	s := &Service{
		cfg: &config{
			chain: &chainMock.ChainService{
				ReceiveBlockMockErr: err,
			},
		},
		seenBlockCache: lruwrpr.New(10),
		badBlockCache:  lruwrpr.New(10),
	}
	require.ErrorIs(t, s.beaconBlockSubscriber(context.Background(), util.NewBeaconBlock()), blockchain.ErrUndefinedExecutionEngineError)
	require.Equal(t, 0, len(s.badBlockCache.Keys()))
	require.Equal(t, 1, len(s.seenBlockCache.Keys()))
}

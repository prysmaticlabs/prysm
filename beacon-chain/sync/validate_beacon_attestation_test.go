package sync

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/prysmaticlabs/go-bitfield"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	lruwrpr "github.com/prysmaticlabs/prysm/cache/lru"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestService_validateCommitteeIndexBeaconAttestation(t *testing.T) {
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	db := dbtest.SetupDB(t)
	chain := &mockChain.ChainService{
		// 1 slot ago.
		Genesis:          time.Now().Add(time.Duration(-1*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second),
		ValidatorsRoot:   [32]byte{'A'},
		ValidAttestation: true,
	}

	s := &Service{
		cfg: &config{
			initialSync:         &mockSync.Sync{IsSyncing: false},
			p2p:                 p,
			beaconDB:            db,
			chain:               chain,
			attestationNotifier: (&mockChain.ChainService{}).OperationNotifier(),
		},
		blkRootToPendingAtts:             make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
		seenUnAggregatedAttestationCache: lruwrpr.New(10),
	}
	s.initCaches()

	invalidRoot := [32]byte{'A', 'B', 'C', 'D'}
	s.setBadBlock(ctx, invalidRoot)

	digest, err := s.currentForkDigest()
	require.NoError(t, err)

	blk := util.NewBeaconBlock()
	blk.Block.Slot = 1
	wsb, err := wrapper.WrappedSignedBeaconBlock(blk)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wsb))

	validBlockRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	chain.FinalizedCheckPoint = &ethpb.Checkpoint{
		Root:  validBlockRoot[:],
		Epoch: 0,
	}

	validators := uint64(64)
	savedState, keys := util.DeterministicGenesisState(t, validators)
	require.NoError(t, savedState.SetSlot(1))
	require.NoError(t, db.SaveState(context.Background(), savedState, validBlockRoot))
	chain.State = savedState

	tests := []struct {
		name                      string
		msg                       *ethpb.Attestation
		topic                     string
		validAttestationSignature bool
		want                      bool
	}{
		{
			name: "valid attestation signature",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b101},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: validBlockRoot[:],
					CommitteeIndex:  0,
					Slot:            1,
					Target: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  validBlockRoot[:],
					},
					Source: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/beacon_attestation_1", digest),
			validAttestationSignature: true,
			want:                      true,
		},
		{
			name: "valid attestation signature with nil topic",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b101},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: validBlockRoot[:],
					CommitteeIndex:  0,
					Slot:            1,
					Target: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  validBlockRoot[:],
					},
					Source: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				},
			},
			topic:                     "",
			validAttestationSignature: true,
			want:                      false,
		},
		{
			name: "bad target epoch",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b101},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: validBlockRoot[:],
					CommitteeIndex:  0,
					Slot:            1,
					Target: &ethpb.Checkpoint{
						Epoch: 10,
						Root:  validBlockRoot[:],
					},
					Source: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/beacon_attestation_1", digest),
			validAttestationSignature: true,
			want:                      false,
		},
		{
			name: "already seen",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b101},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: validBlockRoot[:],
					CommitteeIndex:  0,
					Slot:            1,
					Target:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
					Source:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/beacon_attestation_1", digest),
			validAttestationSignature: true,
			want:                      false,
		},
		{
			name: "invalid beacon block",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b101},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: invalidRoot[:],
					CommitteeIndex:  0,
					Slot:            1,
					Target:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
					Source:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/beacon_attestation_1", digest),
			validAttestationSignature: true,
			want:                      false,
		},
		{
			name: "committee index exceeds committee length",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b101},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: validBlockRoot[:],
					CommitteeIndex:  4,
					Slot:            1,
					Target:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
					Source:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/beacon_attestation_2", digest),
			validAttestationSignature: true,
			want:                      false,
		},
		{
			name: "wrong committee index",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b101},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: validBlockRoot[:],
					CommitteeIndex:  2,
					Slot:            1,
					Target:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
					Source:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/beacon_attestation_2", digest),
			validAttestationSignature: true,
			want:                      false,
		},
		{
			name: "already aggregated",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b1011},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: validBlockRoot[:],
					CommitteeIndex:  1,
					Slot:            1,
					Target:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
					Source:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/beacon_attestation_1", digest),
			validAttestationSignature: true,
			want:                      false,
		},
		{
			name: "missing block",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b101},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: bytesutil.PadTo([]byte("missing"), fieldparams.RootLength),
					CommitteeIndex:  1,
					Slot:            1,
					Target:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
					Source:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/beacon_attestation_1", digest),
			validAttestationSignature: true,
			want:                      false,
		},
		{
			name: "invalid attestation",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b101},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: validBlockRoot[:],
					CommitteeIndex:  1,
					Slot:            1,
					Target:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
					Source:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/beacon_attestation_1", digest),
			validAttestationSignature: false,
			want:                      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.ClearCache()
			chain.ValidAttestation = tt.validAttestationSignature
			if tt.validAttestationSignature {
				com, err := helpers.BeaconCommitteeFromState(context.Background(), savedState, tt.msg.Data.Slot, tt.msg.Data.CommitteeIndex)
				require.NoError(t, err)
				domain, err := signing.Domain(savedState.Fork(), tt.msg.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester, savedState.GenesisValidatorsRoot())
				require.NoError(t, err)
				attRoot, err := signing.ComputeSigningRoot(tt.msg.Data, domain)
				require.NoError(t, err)
				for i := 0; ; i++ {
					if tt.msg.AggregationBits.BitAt(uint64(i)) {
						tt.msg.Signature = keys[com[i]].Sign(attRoot[:]).Marshal()
						break
					}
				}
			} else {
				tt.msg.Signature = make([]byte, 96)
			}
			buf := new(bytes.Buffer)
			_, err := p.Encoding().EncodeGossip(buf, tt.msg)
			require.NoError(t, err)
			m := &pubsub.Message{
				Message: &pubsubpb.Message{
					Data:  buf.Bytes(),
					Topic: &tt.topic,
				},
			}
			if tt.topic == "" {
				m.Message.Topic = nil
			}

			res, err := s.validateCommitteeIndexBeaconAttestation(ctx, "" /*peerID*/, m)
			received := res == pubsub.ValidationAccept
			if received != tt.want {
				t.Fatalf("Did not received wanted validation. Got %v, wanted %v", !tt.want, tt.want)
			}
			if tt.want && err != nil {
				t.Errorf("Non nil error returned: %v", err)
			}
			if tt.want && m.ValidatorData == nil {
				t.Error("Expected validator data to be set")
			}
		})
	}
}

func TestServiceValidateCommitteeIndexBeaconAttestation_Optimistic(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	slashing, s := setupValidAttesterSlashing(t)

	r := &Service{
		cfg: &config{
			p2p:         p,
			chain:       &mockChain.ChainService{State: s, Optimistic: true},
			initialSync: &mockSync.Sync{IsSyncing: false},
		},
	}

	buf := new(bytes.Buffer)
	_, err := p.Encoding().EncodeGossip(buf, slashing)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(slashing)]
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}
	res, err := r.validateCommitteeIndexBeaconAttestation(ctx, "foobar", msg)
	assert.NoError(t, err)
	valid := res == pubsub.ValidationIgnore
	assert.Equal(t, true, valid, "Should have ignore this message")
}

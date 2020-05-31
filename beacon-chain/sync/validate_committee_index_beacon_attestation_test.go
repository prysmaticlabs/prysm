package sync

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestService_validateCommitteeIndexBeaconAttestation(t *testing.T) {
	t.Skip("Temporarily disabled, fixed in v0.12 branch.")

	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	db := dbtest.SetupDB(t)
	chain := &mockChain.ChainService{
		Genesis:          time.Now().Add(time.Duration(-64*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second), // 64 slots ago
		ValidatorsRoot:   [32]byte{'A'},
		ValidAttestation: true,
	}

	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{
		initialSync:          &mockSync.Sync{IsSyncing: false},
		p2p:                  p,
		db:                   db,
		chain:                chain,
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
		seenAttestationCache: c,
		stateSummaryCache:    cache.NewStateSummaryCache(),
	}
	digest, err := s.forkDigest()
	if err != nil {
		t.Fatal(err)
	}

	blk := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot: 55,
		},
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}

	validBlockRoot, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}

	savedState := testutil.NewBeaconState()
	if err := db.SaveState(context.Background(), savedState, validBlockRoot); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name                      string
		msg                       *ethpb.Attestation
		topic                     string
		validAttestationSignature bool
		want                      bool
	}{
		{
			name: "validAttestationSignature",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b1010},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: validBlockRoot[:],
					CommitteeIndex:  1,
					Slot:            63,
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/committee_index1_beacon_attestation", digest),
			validAttestationSignature: true,
			want:                      true,
		},
		{
			name: "alreadySeen",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b1010},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: validBlockRoot[:],
					CommitteeIndex:  1,
					Slot:            63,
					Target:          &ethpb.Checkpoint{},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/committee_index1_beacon_attestation", digest),
			validAttestationSignature: true,
			want:                      false,
		},
		{
			name: "wrong committee index",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b1010},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: validBlockRoot[:],
					CommitteeIndex:  2,
					Slot:            63,
					Target:          &ethpb.Checkpoint{},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/committee_index3_beacon_attestation", digest),
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
					Slot:            63,
					Target:          &ethpb.Checkpoint{},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/committee_index1_beacon_attestation", digest),
			validAttestationSignature: true,
			want:                      false,
		},
		{
			name: "missing block",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b1010},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: bytesutil.PadTo([]byte("missing"), 32),
					CommitteeIndex:  1,
					Slot:            63,
					Target:          &ethpb.Checkpoint{},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/committee_index1_beacon_attestation", digest),
			validAttestationSignature: true,
			want:                      false,
		},
		{
			name: "invalid attestation",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b1010},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: validBlockRoot[:],
					CommitteeIndex:  1,
					Slot:            63,
					Target:          &ethpb.Checkpoint{},
				},
			},
			topic:                     fmt.Sprintf("/eth2/%x/committee_index1_beacon_attestation", digest),
			validAttestationSignature: false,
			want:                      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			_, err := p.Encoding().Encode(buf, tt.msg)
			if err != nil {
				t.Error(err)
			}
			m := &pubsub.Message{
				Message: &pubsubpb.Message{
					Data:     buf.Bytes(),
					TopicIDs: []string{tt.topic},
				},
			}
			received := s.validateCommitteeIndexBeaconAttestation(ctx, "" /*peerID*/, m) == pubsub.ValidationAccept
			if received != tt.want {
				t.Fatalf("Did not received wanted validation. Got %v, wanted %v", !tt.want, tt.want)
			}
			if tt.want && m.ValidatorData == nil {
				t.Error("Expected validator data to be set")
			}
		})
	}
}

package sync

import (
	"bytes"
	"context"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestService_validateCommitteeIndexBeaconAttestation(t *testing.T) {
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	chain := &mockChain.ChainService{
		Genesis:          time.Now().Add(time.Duration(-64*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second), // 64 slots ago
		ValidAttestation: true,
	}
	s := &Service{
		initialSync:          &mockSync.Sync{IsSyncing: false},
		p2p:                  p,
		db:                   db,
		chain:                chain,
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.AggregateAttestationAndProof),
	}

	blk := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot: 55,
		},
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}

	validBlockRoot, err := ssz.HashTreeRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}

	savedState, _ := beaconstate.InitializeFromProto(&pb.BeaconState{})
	db.SaveState(context.Background(), savedState, validBlockRoot)

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
			topic:                     "/eth2/committee_index1_beacon_attestation",
			validAttestationSignature: true,
			want:                      true,
		},
		{
			name: "wrong committee index",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b1010},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: validBlockRoot[:],
					CommitteeIndex:  2,
					Slot:            63,
				},
			},
			topic:                     "/eth2/committee_index3_beacon_attestation",
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
				},
			},
			topic:                     "/eth2/committee_index1_beacon_attestation",
			validAttestationSignature: true,
			want:                      false,
		},
		{
			name: "missing block",
			msg: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b1010},
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: []byte("missing"),
					CommitteeIndex:  1,
					Slot:            63,
				},
			},
			topic:                     "/eth2/committee_index1_beacon_attestation",
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
				},
			},
			topic:                     "/eth2/committee_index1_beacon_attestation",
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
			chain.ValidAttestation = tt.validAttestationSignature
			if s.validateCommitteeIndexBeaconAttestation(ctx, "" /*peerID*/, m) != tt.want {
				t.Errorf("Did not received wanted validation. Got %v, wanted %v", !tt.want, tt.want)
			}
			if tt.want && m.ValidatorData == nil {
				t.Error("Expected validator data to be set")
			}
		})
	}
}

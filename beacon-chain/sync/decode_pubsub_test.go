package sync

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/d4l3k/messagediff"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptesting "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestService_decodePubsubMessage(t *testing.T) {
	digest, err := signing.ComputeForkDigest(params.BeaconConfig().GenesisForkVersion, make([]byte, 32))
	require.NoError(t, err)
	tests := []struct {
		name    string
		topic   string
		input   *pubsub.Message
		want    interface{}
		wantErr error
	}{
		{
			name:    "Nil message",
			input:   nil,
			wantErr: errNilPubsubMessage,
		},
		{
			name: "nil topic",
			input: &pubsub.Message{
				Message: &pb.Message{
					Topic: nil,
				},
			},
			wantErr: errNilPubsubMessage,
		},
		{
			name:    "invalid topic format",
			topic:   "foo",
			wantErr: errInvalidTopic,
		},
		{
			name:    "topic not mapped to any message type",
			topic:   "/eth2/abababab/foo/ssz_snappy",
			wantErr: p2p.ErrMessageNotMapped,
		},
		{
			name:  "valid message -- beacon block",
			topic: fmt.Sprintf(p2p.GossipTypeMapping[reflect.TypeOf(&ethpb.SignedBeaconBlock{})], digest),
			input: &pubsub.Message{
				Message: &pb.Message{
					Data: func() []byte {
						buf := new(bytes.Buffer)
						if _, err := p2ptesting.NewTestP2P(t).Encoding().EncodeGossip(buf, util.NewBeaconBlock()); err != nil {
							t.Fatal(err)
						}
						return buf.Bytes()
					}(),
				},
			},
			wantErr: nil,
			want: func() block.SignedBeaconBlock {
				wsb, err := wrapper.WrappedSignedBeaconBlock(util.NewBeaconBlock())
				require.NoError(t, err)
				return wsb
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				cfg: &config{p2p: p2ptesting.NewTestP2P(t), chain: &mock.ChainService{ValidatorsRoot: [32]byte{}, Genesis: time.Now()}},
			}
			if tt.topic != "" {
				if tt.input == nil {
					tt.input = &pubsub.Message{Message: &pb.Message{}}
				} else if tt.input.Message == nil {
					tt.input.Message = &pb.Message{}
				}
				// reassign because tt is a loop variable
				topic := tt.topic
				tt.input.Message.Topic = &topic
			}
			got, err := s.decodePubsubMessage(tt.input)
			if err != nil && err != tt.wantErr && !strings.Contains(err.Error(), tt.wantErr.Error()) {
				t.Errorf("decodePubsubMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				diff, _ := messagediff.PrettyDiff(got, tt.want)
				t.Log(diff)
				t.Errorf("decodePubsubMessage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

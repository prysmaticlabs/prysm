package sync

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/d4l3k/messagediff"
	"github.com/gogo/protobuf/proto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptesting "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestService_decodePubsubMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   *pubsub.Message
		want    proto.Message
		wantErr error
	}{
		{
			name:    "Nil message",
			input:   nil,
			wantErr: errNilPubsubMessage,
		},
		{
			name: "More than 1 topic ID",
			input: &pubsub.Message{
				Message: &pb.Message{
					TopicIDs: []string{"foo", "bar"},
				},
			},
			wantErr: errTooManyTopics,
		},
		{
			name: "invalid topic format",
			input: &pubsub.Message{
				Message: &pb.Message{
					TopicIDs: []string{"foo"}, // Topic should be in format of /eth2/%x/{something}.
				},
			},
			wantErr: errInvalidTopic,
		},
		{
			name: "topic not mapped to any message type",
			input: &pubsub.Message{
				Message: &pb.Message{
					TopicIDs: []string{"/eth2/abcdef/foo"},
				},
			},
			wantErr: p2p.ErrMessageNotMapped,
		},
		{
			name: "valid message -- beacon block",
			input: &pubsub.Message{
				Message: &pb.Message{
					TopicIDs: []string{p2p.GossipTypeMapping[reflect.TypeOf(&ethpb.SignedBeaconBlock{})]},
					Data: func() []byte {
						buf := new(bytes.Buffer)
						if _, err := p2ptesting.NewTestP2P(t).Encoding().EncodeGossip(buf, testutil.NewBeaconBlock()); err != nil {
							t.Fatal(err)
						}
						return buf.Bytes()
					}(),
				},
			},
			wantErr: nil,
			want:    testutil.NewBeaconBlock(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				p2p: p2ptesting.NewTestP2P(t),
			}
			got, err := s.decodePubsubMessage(tt.input)
			if err != tt.wantErr {
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

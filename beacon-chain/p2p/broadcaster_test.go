package p2p

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	testpb "github.com/prysmaticlabs/prysm/proto/testing"
	p2ptest "github.com/prysmaticlabs/prysm/shared/mock/p2p"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestService_Broadcast(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.Host.Network().Peers()) == 0 {
		t.Fatal("No peers")
	}

	p := &Service{
		host:   p1.Host,
		pubsub: p1.PubSub(),
		cfg: &Config{
			Encoding: "ssz",
		},
		genesisTime:           time.Now(),
		genesisValidatorsRoot: []byte{'A'},
	}

	msg := &testpb.TestSimpleMessage{
		Bar: 55,
	}

	topic := "/eth2/%x/testing"
	// Set a test gossip mapping for testpb.TestSimpleMessage.
	GossipTypeMapping[reflect.TypeOf(msg)] = topic
	digest, err := p.forkDigest()
	if err != nil {
		t.Fatal(err)
	}
	topic = fmt.Sprintf(topic, digest)

	// External peer subscribes to the topic.
	topic += p.Encoding().ProtocolSuffix()
	sub, err := p2.PubSub().Subscribe(topic)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(50 * time.Millisecond) // libp2p fails without this delay...

	// Async listen for the pubsub, must be before the broadcast.
	var wg sync.WaitGroup
	wg.Add(1)
	go func(tt *testing.T) {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		incomingMessage, err := sub.Next(ctx)
		if err != nil {
			tt.Fatal(err)
		}

		result := &testpb.TestSimpleMessage{}
		if err := p.Encoding().DecodeGossip(incomingMessage.Data, result); err != nil {
			tt.Fatal(err)
		}
		if !proto.Equal(result, msg) {
			tt.Errorf("Did not receive expected message, got %+v, wanted %+v", result, msg)
		}
	}(t)

	// Broadcast to peers and wait.
	if err := p.Broadcast(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Error("Failed to receive pubsub within 1s")
	}
}

func TestService_Broadcast_ReturnsErr_TopicNotMapped(t *testing.T) {
	p := Service{
		genesisTime:           time.Now(),
		genesisValidatorsRoot: []byte{'A'},
	}
	if err := p.Broadcast(context.Background(), &testpb.AddressBook{}); err != ErrMessageNotMapped {
		t.Fatalf("Expected error %v, got %v", ErrMessageNotMapped, err)
	}
}

func TestService_Attestation_Subnet(t *testing.T) {
	if gtm := GossipTypeMapping[reflect.TypeOf(&eth.Attestation{})]; gtm != attestationSubnetTopicFormat {
		t.Errorf("Constant is out of date. Wanted %s, got %s", attestationSubnetTopicFormat, gtm)
	}

	tests := []struct {
		att   *eth.Attestation
		topic string
	}{
		{
			att: &eth.Attestation{
				Data: &eth.AttestationData{
					CommitteeIndex: 0,
				},
			},
			topic: "/eth2/00000000/committee_index0_beacon_attestation",
		},
		{
			att: &eth.Attestation{
				Data: &eth.AttestationData{
					CommitteeIndex: 11,
				},
			},
			topic: "/eth2/00000000/committee_index11_beacon_attestation",
		},
		{
			att: &eth.Attestation{
				Data: &eth.AttestationData{
					CommitteeIndex: 55,
				},
			},
			topic: "/eth2/00000000/committee_index55_beacon_attestation",
		},
		{
			att:   &eth.Attestation{},
			topic: "",
		},
		{
			topic: "",
		},
	}
	for _, tt := range tests {
		if res := attestationToTopic(tt.att, [4]byte{} /* fork digest */); res != tt.topic {
			t.Errorf("Wrong topic, got %s wanted %s", res, tt.topic)
		}
	}
}

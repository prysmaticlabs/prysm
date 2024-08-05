package sync

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/pkg/errors"
	mock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/startup"
	mockSync "github.com/prysmaticlabs/prysm/v5/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/verification"
	lruwrpr "github.com/prysmaticlabs/prysm/v5/cache/lru"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	// payloadattestation "github.com/prysmaticlabs/prysm/v5/consensus-types/epbs/payload-attestation"
	// ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestValidatePayloadAttestation_FromSelf(t *testing.T) {
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	s := &Service{cfg: &config{p2p: p}}

	result, err := s.validatePayloadAttestation(ctx, s.cfg.p2p.PeerID(), nil)
	require.NoError(t, err)
	require.Equal(t, result, pubsub.ValidationAccept)
}

func TestValidatePayloadAttestation_InitSync(t *testing.T) {
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	s := &Service{cfg: &config{p2p: p, initialSync: &mockSync.Sync{IsSyncing: true}}}
	result, err := s.validatePayloadAttestation(ctx, "", nil)
	require.NoError(t, err)
	require.Equal(t, result, pubsub.ValidationIgnore)
}

func TestValidatePayloadAttestation_InvalidTopic(t *testing.T) {
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	s := &Service{cfg: &config{p2p: p, initialSync: &mockSync.Sync{}}}
	result, err := s.validatePayloadAttestation(ctx, "", &pubsub.Message{
		Message: &pb.Message{},
	})
	require.ErrorIs(t, errInvalidTopic, err)
	require.Equal(t, result, pubsub.ValidationReject)
}

func TestValidatePayloadAttestation_InvalidMessageType(t *testing.T) {
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	chainService := &mock.ChainService{Genesis: time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0)}
	s := &Service{cfg: &config{
		p2p: p, 
		initialSync: &mockSync.Sync{}, 
		clock: startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot)}}

	msg := util.NewBeaconBlock()
	buf := new(bytes.Buffer)
	_, err := p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(msg)]
	digest, err := s.currentForkDigest()
	require.NoError(t, err)
	topic = s.addDigestToTopic(topic, digest)
	result, err := s.validatePayloadAttestation(ctx, "", &pubsub.Message{
		Message: &pb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		}})
	require.ErrorIs(t, errWrongMessage, err)
	require.Equal(t, result, pubsub.ValidationReject)
}

func TestValidatePayloadAttestation_ErrorPathsWithMock(t *testing.T) {
	tests := []struct {
		name     string
		error    error
		verifier verification.PayloadAttestationMsgVerifier
		result   pubsub.ValidationResult
	}{
		{
			name : "",
			error: errors.New("payload att slot does not match the current slot"),
			verifier:  	&verification.MockPayloadAttestation{ErrIncorrectPayloadAttSlot: errors.New("payload att slot does not match the current slot")},
			result: pubsub.ValidationReject,
		},
		{
			name : "",
			error: errors.New("unknown payload att status"),
			verifier:  	&verification.MockPayloadAttestation{ErrIncorrectPayloadAttStatus: errors.New("unknown payload att status")},			
			result: pubsub.ValidationReject,
		},
		{
			name : "",
			error: errors.New("block root not seen"),
			verifier:  	&verification.MockPayloadAttestation{ErrPayloadAttBlockRootNotSeen: errors.New("block root not seen")},		
			result: pubsub.ValidationReject,
		},
		{
			name : "",
			error: errors.New("block root invalid"),
			verifier:  	&verification.MockPayloadAttestation{ErrPayloadAttBlockRootInvalid: errors.New("block root invalid")},		
			result: pubsub.ValidationReject,
		},
		{
			name : "",
			error: errors.New("validator not present in payload timeliness committee"),
			verifier:  	&verification.MockPayloadAttestation{ErrIncorrectPayloadAttValidator: errors.New("validator not present in payload timeliness committee")},		
			result: pubsub.ValidationReject,
		},
		{
			name : "",
			error: errors.New("invalid payload attestation message"),
			verifier:  	&verification.MockPayloadAttestation{ErrInvalidPayloadAttMessage: errors.New("invalid payload attestation message")},		
			result: pubsub.ValidationReject,
		},
	}
	for _, tt := range tests {
		t.Run(tt.error.Error(), func(t *testing.T) {
			ctx := context.Background()//ok
			p := p2ptest.NewTestP2P(t)//ok
			chainService := &mock.ChainService{Genesis: time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0)}  //doubt
			s := &Service{ //ok
				seenBlobCache:     lruwrpr.New(10), //doubt
				seenPendingBlocks: make(map[[32]byte]bool), //doubt
				cfg:               &config{chain: chainService, p2p: p, initialSync: &mockSync.Sync{}, clock: startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot)}}
			s.PayloadAttestationMsgVerifier=tt.verifier
			gs, pk := util.DeterministicGenesisState(t, 32)
				scs, err := util.GenerateAttestations(gs, pk, 1, params.BeaconConfig().SlotsPerEpoch, false)
				msg := scs[0]
				buf := new(bytes.Buffer)
			_, err = p.Encoding().EncodeGossip(buf, msg)
			require.NoError(t, err)
			topic := p2p.GossipTypeMapping[reflect.TypeOf(msg)]
			digest, err := s.currentForkDigest()
			require.NoError(t, err)
			topic = s.addDigestAndIndexToTopic(topic, digest, 0)
			result, err := s.validatePayloadAttestation(ctx, "", &pubsub.Message{
				Message: &pb.Message{
					Data:  buf.Bytes(),
					Topic: &topic,
				}})
			require.ErrorContains(t, tt.error.Error(), err)
			require.Equal(t,tt.result,result)
		})
	}
}

func testPayloadAttestationVerifier() verification.PayloadAttestationMsgVerifier {
		return &verification.MockPayloadAttestation{}
}

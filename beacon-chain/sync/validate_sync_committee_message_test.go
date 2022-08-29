package sync

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang/snappy"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	mockChain "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	testingdb "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/encoder"
	mockp2p "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/testing"
	p2ptypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/v3/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

func TestService_ValidateSyncCommitteeMessage(t *testing.T) {
	beaconDB := testingdb.SetupDB(t)
	headRoot, keys := fillUpBlocksAndState(context.Background(), t, beaconDB)
	defaultTopic := p2p.SyncCommitteeSubnetTopicFormat
	fakeDigest := []byte{0xAB, 0x00, 0xCC, 0x9E}
	defaultTopic = defaultTopic + "/" + encoder.ProtocolSuffixSSZSnappy
	chainService := &mockChain.ChainService{
		Genesis:        time.Now(),
		ValidatorsRoot: [32]byte{'A'},
	}
	emptySig := [96]byte{}
	type args struct {
		ctx   context.Context
		pid   peer.ID
		msg   *ethpb.SyncCommitteeMessage
		topic string
	}
	tests := []struct {
		name     string
		svc      *Service
		setupSvc func(s *Service, msg *ethpb.SyncCommitteeMessage, topic string) (*Service, string)
		args     args
		want     pubsub.ValidationResult
	}{
		{
			name: "Is syncing",
			svc: NewService(context.Background(),
				WithP2P(mockp2p.NewTestP2P(t)),
				WithInitialSync(&mockSync.Sync{IsSyncing: true}),
				WithChainService(chainService),
				WithStateNotifier(chainService.StateNotifier()),
				WithOperationNotifier(chainService.OperationNotifier()),
			),
			setupSvc: func(s *Service, msg *ethpb.SyncCommitteeMessage, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB)
				msg.BlockRoot = headRoot[:]
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				return s, topic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: "junk",
				msg: &ethpb.SyncCommitteeMessage{
					Slot:           1,
					ValidatorIndex: 1,
					BlockRoot:      params.BeaconConfig().ZeroHash[:],
					Signature:      emptySig[:],
				}},
			want: pubsub.ValidationIgnore,
		},
		{
			name: "Bad Topic",
			svc: NewService(context.Background(),
				WithP2P(mockp2p.NewTestP2P(t)),
				WithInitialSync(&mockSync.Sync{IsSyncing: false}),
				WithChainService(chainService),
				WithStateNotifier(chainService.StateNotifier()),
				WithOperationNotifier(chainService.OperationNotifier()),
			),
			setupSvc: func(s *Service, msg *ethpb.SyncCommitteeMessage, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB)
				msg.BlockRoot = headRoot[:]
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				return s, topic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: "junk",
				msg: &ethpb.SyncCommitteeMessage{
					Slot:           1,
					ValidatorIndex: 1,
					BlockRoot:      params.BeaconConfig().ZeroHash[:],
					Signature:      emptySig[:],
				}},
			want: pubsub.ValidationReject,
		},
		{
			name: "Future Slot Message",
			svc: NewService(context.Background(),
				WithP2P(mockp2p.NewTestP2P(t)),
				WithInitialSync(&mockSync.Sync{IsSyncing: false}),
				WithChainService(chainService),
				WithStateNotifier(chainService.StateNotifier()),
				WithOperationNotifier(chainService.OperationNotifier()),
			),
			setupSvc: func(s *Service, msg *ethpb.SyncCommitteeMessage, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB)
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				return s, topic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: fmt.Sprintf(defaultTopic, fakeDigest, 0),
				msg: &ethpb.SyncCommitteeMessage{
					Slot:           10,
					ValidatorIndex: 1,
					BlockRoot:      params.BeaconConfig().ZeroHash[:],
					Signature:      emptySig[:],
				}},
			want: pubsub.ValidationIgnore,
		},
		{
			name: "Already Seen Message",
			svc: NewService(context.Background(),
				WithP2P(mockp2p.NewTestP2P(t)),
				WithInitialSync(&mockSync.Sync{IsSyncing: false}),
				WithChainService(chainService),
				WithStateNotifier(chainService.StateNotifier()),
				WithOperationNotifier(chainService.OperationNotifier()),
			),
			setupSvc: func(s *Service, msg *ethpb.SyncCommitteeMessage, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB)
				s.cfg.beaconDB = beaconDB
				s.initCaches()

				s.setSeenSyncMessageIndexSlot(1, 1, 0)
				return s, topic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: fmt.Sprintf(defaultTopic, fakeDigest, 0),
				msg: &ethpb.SyncCommitteeMessage{
					Slot:           1,
					ValidatorIndex: 1,
					BlockRoot:      params.BeaconConfig().ZeroHash[:],
					Signature:      emptySig[:],
				}},
			want: pubsub.ValidationIgnore,
		},
		{
			name: "Non-existent block root",
			svc: NewService(context.Background(),
				WithP2P(mockp2p.NewTestP2P(t)),
				WithInitialSync(&mockSync.Sync{IsSyncing: false}),
				WithChainService(chainService),
				WithStateNotifier(chainService.StateNotifier()),
				WithOperationNotifier(chainService.OperationNotifier()),
			),
			setupSvc: func(s *Service, msg *ethpb.SyncCommitteeMessage, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB)
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				s.cfg.chain = &mockChain.ChainService{
					ValidatorsRoot: [32]byte{'A'},
					Genesis:        time.Now().Add(-time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Duration(10)),
				}
				incorrectRoot := [32]byte{0xBB}
				msg.BlockRoot = incorrectRoot[:]

				return s, topic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: fmt.Sprintf(defaultTopic, fakeDigest, 0),
				msg: &ethpb.SyncCommitteeMessage{
					Slot:           1,
					ValidatorIndex: 1,
					BlockRoot:      params.BeaconConfig().ZeroHash[:],
					Signature:      emptySig[:],
				}},
			want: pubsub.ValidationIgnore,
		},
		{
			name: "Subnet is non-existent",
			svc: NewService(context.Background(),
				WithP2P(mockp2p.NewTestP2P(t)),
				WithInitialSync(&mockSync.Sync{IsSyncing: false}),
				WithChainService(chainService),
				WithStateNotifier(chainService.StateNotifier()),
				WithOperationNotifier(chainService.OperationNotifier()),
			),
			setupSvc: func(s *Service, msg *ethpb.SyncCommitteeMessage, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB)
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				msg.BlockRoot = headRoot[:]
				hState, err := beaconDB.State(context.Background(), headRoot)
				assert.NoError(t, err)
				s.cfg.chain = &mockChain.ChainService{
					SyncCommitteeIndices: []types.CommitteeIndex{0},
					ValidatorsRoot:       [32]byte{'A'},
					Genesis:              time.Now().Add(-time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Duration(hState.Slot()-1)),
				}
				numOfVals := hState.NumValidators()

				chosenVal := numOfVals - 10
				msg.Signature = emptySig[:]
				msg.BlockRoot = headRoot[:]
				msg.ValidatorIndex = types.ValidatorIndex(chosenVal)
				msg.Slot = slots.PrevSlot(hState.Slot())

				// Set Bad Topic and Subnet
				digest, err := s.currentForkDigest()
				assert.NoError(t, err)
				actualTopic := fmt.Sprintf(defaultTopic, digest, 5)

				return s, actualTopic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: defaultTopic,
				msg: &ethpb.SyncCommitteeMessage{
					Slot:           1,
					ValidatorIndex: 1,
					BlockRoot:      params.BeaconConfig().ZeroHash[:],
					Signature:      emptySig[:],
				}},
			want: pubsub.ValidationReject,
		},
		{
			name: "Validator is non-existent",
			svc: NewService(context.Background(),
				WithP2P(mockp2p.NewTestP2P(t)),
				WithInitialSync(&mockSync.Sync{IsSyncing: false}),
				WithChainService(chainService),
				WithStateNotifier(chainService.StateNotifier()),
				WithOperationNotifier(chainService.OperationNotifier()),
			),
			setupSvc: func(s *Service, msg *ethpb.SyncCommitteeMessage, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB)
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				msg.BlockRoot = headRoot[:]
				hState, err := beaconDB.State(context.Background(), headRoot)
				assert.NoError(t, err)
				s.cfg.chain = &mockChain.ChainService{
					ValidatorsRoot: [32]byte{'A'},
					Genesis:        time.Now().Add(-time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Duration(hState.Slot()-1)),
				}

				numOfVals := hState.NumValidators()

				chosenVal := numOfVals + 10
				msg.Signature = emptySig[:]
				msg.BlockRoot = headRoot[:]
				msg.ValidatorIndex = types.ValidatorIndex(chosenVal)
				msg.Slot = slots.PrevSlot(hState.Slot())

				digest, err := s.currentForkDigest()
				assert.NoError(t, err)
				actualTopic := fmt.Sprintf(defaultTopic, digest, 1)

				return s, actualTopic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: defaultTopic,
				msg: &ethpb.SyncCommitteeMessage{
					Slot:           1,
					ValidatorIndex: 1,
					BlockRoot:      params.BeaconConfig().ZeroHash[:],
					Signature:      emptySig[:],
				}},
			want: pubsub.ValidationIgnore,
		},
		{
			name: "Invalid Sync Committee Signature",
			svc: NewService(context.Background(),
				WithP2P(mockp2p.NewTestP2P(t)),
				WithInitialSync(&mockSync.Sync{IsSyncing: false}),
				WithChainService(chainService),
				WithStateNotifier(chainService.StateNotifier()),
				WithOperationNotifier(chainService.OperationNotifier()),
			),
			setupSvc: func(s *Service, msg *ethpb.SyncCommitteeMessage, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB)
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				msg.BlockRoot = headRoot[:]
				hState, err := beaconDB.State(context.Background(), headRoot)
				assert.NoError(t, err)

				numOfVals := hState.NumValidators()

				chosenVal := numOfVals - 10
				msg.Signature = emptySig[:]
				msg.BlockRoot = headRoot[:]
				msg.ValidatorIndex = types.ValidatorIndex(chosenVal)
				msg.Slot = slots.PrevSlot(hState.Slot())

				d, err := signing.Domain(hState.Fork(), slots.ToEpoch(hState.Slot()), params.BeaconConfig().DomainSyncCommittee, hState.GenesisValidatorsRoot())
				assert.NoError(t, err)
				subCommitteeSize := params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount
				s.cfg.chain = &mockChain.ChainService{
					SyncCommitteeIndices: []types.CommitteeIndex{types.CommitteeIndex(subCommitteeSize)},
					ValidatorsRoot:       [32]byte{'A'},
					Genesis:              time.Now().Add(-time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Duration(hState.Slot()-1)),
					SyncCommitteeDomain:  d,
					PublicKey:            bytesutil.ToBytes48(keys[chosenVal].PublicKey().Marshal()),
				}

				// Set Topic and Subnet
				digest, err := s.currentForkDigest()
				assert.NoError(t, err)
				actualTopic := fmt.Sprintf(defaultTopic, digest, 1)

				return s, actualTopic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: defaultTopic,
				msg: &ethpb.SyncCommitteeMessage{
					Slot:           1,
					ValidatorIndex: 1,
					BlockRoot:      params.BeaconConfig().ZeroHash[:],
					Signature:      emptySig[:],
				}},
			want: pubsub.ValidationReject,
		},
		{
			name: "Valid Sync Committee Signature",
			svc: NewService(context.Background(),
				WithP2P(mockp2p.NewTestP2P(t)),
				WithInitialSync(&mockSync.Sync{IsSyncing: false}),
				WithChainService(chainService),
				WithStateNotifier(chainService.StateNotifier()),
				WithOperationNotifier(chainService.OperationNotifier()),
			),
			setupSvc: func(s *Service, msg *ethpb.SyncCommitteeMessage, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB)
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				msg.BlockRoot = headRoot[:]
				hState, err := beaconDB.State(context.Background(), headRoot)
				assert.NoError(t, err)
				subCommitteeSize := params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount

				numOfVals := hState.NumValidators()

				chosenVal := numOfVals - 10
				d, err := signing.Domain(hState.Fork(), slots.ToEpoch(hState.Slot()), params.BeaconConfig().DomainSyncCommittee, hState.GenesisValidatorsRoot())
				assert.NoError(t, err)
				rawBytes := p2ptypes.SSZBytes(headRoot[:])
				sigRoot, err := signing.ComputeSigningRoot(&rawBytes, d)
				assert.NoError(t, err)

				s.cfg.chain = &mockChain.ChainService{
					SyncCommitteeIndices: []types.CommitteeIndex{types.CommitteeIndex(subCommitteeSize)},
					ValidatorsRoot:       [32]byte{'A'},
					Genesis:              time.Now().Add(-time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Duration(hState.Slot()-1)),
					SyncCommitteeDomain:  d,
					PublicKey:            bytesutil.ToBytes48(keys[chosenVal].PublicKey().Marshal()),
				}

				msg.Signature = keys[chosenVal].Sign(sigRoot[:]).Marshal()
				msg.BlockRoot = headRoot[:]
				msg.ValidatorIndex = types.ValidatorIndex(chosenVal)
				msg.Slot = slots.PrevSlot(hState.Slot())

				// Set Topic and Subnet
				digest, err := s.currentForkDigest()
				assert.NoError(t, err)
				actualTopic := fmt.Sprintf(defaultTopic, digest, 1)

				return s, actualTopic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: defaultTopic,
				msg: &ethpb.SyncCommitteeMessage{
					Slot:           1,
					ValidatorIndex: 1,
					BlockRoot:      params.BeaconConfig().ZeroHash[:],
					Signature:      emptySig[:],
				}},
			want: pubsub.ValidationAccept,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.svc, tt.args.topic = tt.setupSvc(tt.svc, tt.args.msg, tt.args.topic)
			marshalledObj, err := tt.args.msg.MarshalSSZ()
			assert.NoError(t, err)
			marshalledObj = snappy.Encode(nil, marshalledObj)
			msg := &pubsub.Message{
				Message: &pubsubpb.Message{
					Data:  marshalledObj,
					Topic: &tt.args.topic,
				},
				ReceivedFrom:  "",
				ValidatorData: nil,
			}
			if got, err := tt.svc.validateSyncCommitteeMessage(tt.args.ctx, tt.args.pid, msg); got != tt.want {
				_ = err
				t.Errorf("validateSyncCommitteeMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_ignoreHasSeenSyncMsg(t *testing.T) {
	tests := []struct {
		name      string
		setupSvc  func(s *Service, msg *ethpb.SyncCommitteeMessage, topic string) (*Service, string)
		msg       *ethpb.SyncCommitteeMessage
		committee []types.CommitteeIndex
		want      pubsub.ValidationResult
	}{
		{
			name: "has seen",
			setupSvc: func(s *Service, msg *ethpb.SyncCommitteeMessage, topic string) (*Service, string) {
				s.initCaches()
				s.setSeenSyncMessageIndexSlot(1, 0, 0)
				return s, ""
			},
			msg:       &ethpb.SyncCommitteeMessage{ValidatorIndex: 0, Slot: 1},
			committee: []types.CommitteeIndex{1, 2, 3},
			want:      pubsub.ValidationIgnore,
		},
		{
			name: "has not seen",
			setupSvc: func(s *Service, msg *ethpb.SyncCommitteeMessage, topic string) (*Service, string) {
				s.initCaches()
				s.setSeenSyncMessageIndexSlot(1, 0, 0)
				return s, ""
			},
			msg:       &ethpb.SyncCommitteeMessage{ValidatorIndex: 1, Slot: 1},
			committee: []types.CommitteeIndex{1, 2, 3},
			want:      pubsub.ValidationAccept,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{}
			s, _ = tt.setupSvc(s, tt.msg, "")
			f := s.ignoreHasSeenSyncMsg(tt.msg, tt.committee)
			result, err := f(context.Background())
			_ = err
			require.Equal(t, tt.want, result)
		})
	}
}

func TestService_rejectIncorrectSyncCommittee(t *testing.T) {
	tests := []struct {
		name             string
		cfg              *config
		setupTopic       func(s *Service) string
		committeeIndices []types.CommitteeIndex
		want             pubsub.ValidationResult
	}{
		{
			name: "invalid",
			cfg: &config{
				chain: &mockChain.ChainService{
					Genesis:        time.Now(),
					ValidatorsRoot: [32]byte{1},
				},
			},
			committeeIndices: []types.CommitteeIndex{0},
			setupTopic: func(_ *Service) string {
				return "foobar"
			},
			want: pubsub.ValidationReject,
		},
		{
			name: "valid",
			cfg: &config{
				chain: &mockChain.ChainService{
					Genesis:        time.Now(),
					ValidatorsRoot: [32]byte{1},
				},
			},
			committeeIndices: []types.CommitteeIndex{0},
			setupTopic: func(s *Service) string {
				format := p2p.GossipTypeMapping[reflect.TypeOf(&ethpb.SyncCommitteeMessage{})]
				digest, err := s.currentForkDigest()
				require.NoError(t, err)
				prefix := fmt.Sprintf(format, digest, 0 /* validator index 0 */)
				topic := prefix + "foobar"
				return topic
			},
			want: pubsub.ValidationAccept,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				cfg: tt.cfg,
			}
			topic := tt.setupTopic(s)
			f := s.rejectIncorrectSyncCommittee(tt.committeeIndices, topic)
			result, err := f(context.Background())
			_ = err
			require.Equal(t, tt.want, result)
		})
	}
}

func Test_ignoreEmptyCommittee(t *testing.T) {
	tests := []struct {
		name      string
		committee []types.CommitteeIndex
		want      pubsub.ValidationResult
	}{
		{
			name:      "nil",
			committee: nil,
			want:      pubsub.ValidationIgnore,
		},
		{
			name:      "empty",
			committee: []types.CommitteeIndex{},
			want:      pubsub.ValidationIgnore,
		},
		{
			name:      "non-empty",
			committee: []types.CommitteeIndex{1},
			want:      pubsub.ValidationAccept,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := ignoreEmptyCommittee(tt.committee)
			result, err := f(context.Background())
			_ = err
			require.Equal(t, tt.want, result)
		})
	}
}

package sync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/snappy"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/peer"
	mockChain "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	testingdb "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/blstoexec"
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
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

func TestService_ValidateBlsToExecutionChange(t *testing.T) {
	beaconDB := testingdb.SetupDB(t)
	headRoot, keys := fillUpBlocksAndState(context.Background(), t, beaconDB)
	defaultTopic := p2p.BlsToExecutionChangeSubnetTopicFormat
	fakeDigest := []byte{0xAB, 0x00, 0xCC, 0x9E}
	//wantedExecAddress := []byte{0xd8, 0xdA, 0x6B, 0xF2, 0x69, 0x64, 0xaF, 0x9D, 0x7e, 0xEd, 0x9e, 0x03, 0xE5, 0x34, 0x15, 0xD3, 0x7a, 0xA9, 0x60, 0x45}
	defaultTopic = defaultTopic + "/" + encoder.ProtocolSuffixSSZSnappy
	chainService := &mockChain.ChainService{
		Genesis:        time.Now(),
		ValidatorsRoot: [32]byte{'A'},
	}
	emptySig := [96]byte{}
	type args struct {
		ctx   context.Context
		pid   peer.ID
		msg   *ethpb.SignedBLSToExecutionChange
		topic string
	}
	tests := []struct {
		name     string
		svc      *Service
		setupSvc func(s *Service, msg *ethpb.SignedBLSToExecutionChange, topic string) (*Service, string)
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
			setupSvc: func(s *Service, msg *ethpb.SignedBLSToExecutionChange, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB, doublylinkedtree.New())
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				return s, topic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: "junk",
				msg: &ethpb.SignedBLSToExecutionChange{
					Message: &ethpb.BLSToExecutionChange{
						ValidatorIndex:     0,
						FromBlsPubkey:      make([]byte, 48),
						ToExecutionAddress: make([]byte, 20),
					},
					Signature: emptySig[:],
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
			setupSvc: func(s *Service, msg *ethpb.SignedBLSToExecutionChange, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB, doublylinkedtree.New())
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				return s, topic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: "junk",
				msg: &ethpb.SignedBLSToExecutionChange{
					Message: &ethpb.BLSToExecutionChange{
						ValidatorIndex:     0,
						FromBlsPubkey:      make([]byte, 48),
						ToExecutionAddress: make([]byte, 20),
					},
					Signature: emptySig[:],
				}},
			want: pubsub.ValidationReject,
		},
		{
			name: "Already Seen Message",
			svc: NewService(context.Background(),
				WithP2P(mockp2p.NewTestP2P(t)),
				WithInitialSync(&mockSync.Sync{IsSyncing: false}),
				WithChainService(chainService),
				WithStateNotifier(chainService.StateNotifier()),
				WithOperationNotifier(chainService.OperationNotifier()),
				WithBlsToExecPool(blstoexec.NewPool()),
			),
			setupSvc: func(s *Service, msg *ethpb.SignedBLSToExecutionChange, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB, doublylinkedtree.New())
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				s.cfg.blsToExecPool.InsertBLSToExecChange(&ethpb.SignedBLSToExecutionChange{
					Message: &ethpb.BLSToExecutionChange{
						ValidatorIndex:     10,
						FromBlsPubkey:      make([]byte, 48),
						ToExecutionAddress: make([]byte, 20),
					},
					Signature: emptySig[:],
				})
				return s, topic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: fmt.Sprintf(defaultTopic, fakeDigest),
				msg: &ethpb.SignedBLSToExecutionChange{
					Message: &ethpb.BLSToExecutionChange{
						ValidatorIndex:     10,
						FromBlsPubkey:      make([]byte, 48),
						ToExecutionAddress: make([]byte, 20),
					},
					Signature: emptySig[:],
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
			setupSvc: func(s *Service, msg *ethpb.SignedBLSToExecutionChange, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB, doublylinkedtree.New())
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				s.cfg.chain = &mockChain.ChainService{
					ValidatorsRoot: [32]byte{'A'},
					Genesis:        time.Now().Add(-time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Duration(10)),
				}
				incorrectRoot := [32]byte{0xBB}
				//msg.BlockRoot = incorrectRoot[:]
				_ = incorrectRoot
				return s, topic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: fmt.Sprintf(defaultTopic, fakeDigest, 0),
				msg: &ethpb.SignedBLSToExecutionChange{
					Message: &ethpb.BLSToExecutionChange{
						ValidatorIndex:     0,
						FromBlsPubkey:      make([]byte, 48),
						ToExecutionAddress: make([]byte, 20),
					},
					Signature: emptySig[:],
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
			setupSvc: func(s *Service, msg *ethpb.SignedBLSToExecutionChange, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB, doublylinkedtree.New())
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				//msg.BlockRoot = headRoot[:]
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
				//msg.BlockRoot = headRoot[:]
				//msg.ValidatorIndex = types.ValidatorIndex(chosenVal)
				//msg.Slot = slots.PrevSlot(hState.Slot())
				_ = chosenVal
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
				msg: &ethpb.SignedBLSToExecutionChange{
					Message: &ethpb.BLSToExecutionChange{
						ValidatorIndex:     0,
						FromBlsPubkey:      make([]byte, 48),
						ToExecutionAddress: make([]byte, 20),
					},
					Signature: emptySig[:],
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
			setupSvc: func(s *Service, msg *ethpb.SignedBLSToExecutionChange, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB, doublylinkedtree.New())
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				//msg.BlockRoot = headRoot[:]
				hState, err := beaconDB.State(context.Background(), headRoot)
				assert.NoError(t, err)
				s.cfg.chain = &mockChain.ChainService{
					ValidatorsRoot: [32]byte{'A'},
					Genesis:        time.Now().Add(-time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Duration(hState.Slot()-1)),
				}

				numOfVals := hState.NumValidators()

				chosenVal := numOfVals + 10
				msg.Signature = emptySig[:]
				//msg.BlockRoot = headRoot[:]
				//msg.ValidatorIndex = types.ValidatorIndex(chosenVal)
				//msg.Slot = slots.PrevSlot(hState.Slot())
				_ = chosenVal
				digest, err := s.currentForkDigest()
				assert.NoError(t, err)
				actualTopic := fmt.Sprintf(defaultTopic, digest, 1)

				return s, actualTopic
			},
			args: args{
				ctx:   context.Background(),
				pid:   "random",
				topic: defaultTopic,
				msg: &ethpb.SignedBLSToExecutionChange{
					Message: &ethpb.BLSToExecutionChange{
						ValidatorIndex:     0,
						FromBlsPubkey:      make([]byte, 48),
						ToExecutionAddress: make([]byte, 20),
					},
					Signature: emptySig[:],
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
			setupSvc: func(s *Service, msg *ethpb.SignedBLSToExecutionChange, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB, doublylinkedtree.New())
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				//msg.BlockRoot = headRoot[:]
				hState, err := beaconDB.State(context.Background(), headRoot)
				assert.NoError(t, err)

				numOfVals := hState.NumValidators()

				chosenVal := numOfVals - 10
				msg.Signature = emptySig[:]
				//msg.BlockRoot = headRoot[:]
				//msg.ValidatorIndex = types.ValidatorIndex(chosenVal)
				//msg.Slot = slots.PrevSlot(hState.Slot())

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
				msg: &ethpb.SignedBLSToExecutionChange{
					Message: &ethpb.BLSToExecutionChange{
						ValidatorIndex:     0,
						FromBlsPubkey:      make([]byte, 48),
						ToExecutionAddress: make([]byte, 20),
					},
					Signature: emptySig[:],
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
			setupSvc: func(s *Service, msg *ethpb.SignedBLSToExecutionChange, topic string) (*Service, string) {
				s.cfg.stateGen = stategen.New(beaconDB, doublylinkedtree.New())
				s.cfg.beaconDB = beaconDB
				s.initCaches()
				//		msg.BlockRoot = headRoot[:]
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
				//msg.BlockRoot = headRoot[:]
				//msg.ValidatorIndex = types.ValidatorIndex(chosenVal)
				//msg.Slot = slots.PrevSlot(hState.Slot())

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
				msg: &ethpb.SignedBLSToExecutionChange{
					Message: &ethpb.BLSToExecutionChange{
						ValidatorIndex:     0,
						FromBlsPubkey:      make([]byte, 48),
						ToExecutionAddress: make([]byte, 20),
					},
					Signature: emptySig[:],
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
			if got, err := tt.svc.validateBlsToExecutionChange(tt.args.ctx, tt.args.pid, msg); got != tt.want {
				_ = err
				t.Errorf("validateBlsToExecutionChange() = %v, want %v", got, tt.want)
			}
		})
	}
}

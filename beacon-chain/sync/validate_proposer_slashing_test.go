package sync

import (
	"bytes"
	"context"
	"crypto/rand"
	"reflect"
	"testing"
	"time"

	"github.com/prysmaticlabs/go-bitfield"

	lru "github.com/hashicorp/golang-lru"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func setupValidProposerSlashing(t *testing.T) (*ethpb.ProposerSlashing, *stateTrie.BeaconState) {
	validators := make([]*ethpb.Validator, 100)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
			Slashed:           false,
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
			ActivationEpoch:   0,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	currentSlot := uint64(0)
	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: validators,
		Slot:       currentSlot,
		Balances:   validatorBalances,
		Fork: &pb.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			Epoch:           0,
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),

		StateRoots:        make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot),
		BlockRoots:        make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot),
		LatestBlockHeader: &ethpb.BeaconBlockHeader{},
	})
	if err != nil {
		t.Fatal(err)
	}

	domain, err := helpers.Domain(
		state.Fork(),
		helpers.CurrentEpoch(state),
		params.BeaconConfig().DomainBeaconProposer,
		state.GenesisValidatorRoot(),
	)
	if err != nil {
		t.Fatal(err)
	}
	privKey := bls.RandKey()

	someRoot := [32]byte{1, 2, 3}
	someRoot2 := [32]byte{4, 5, 6}
	header1 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: 1,
			Slot:          0,
			ParentRoot:    someRoot[:],
			StateRoot:     someRoot[:],
			BodyRoot:      someRoot[:],
		},
	}
	signingRoot, err := helpers.ComputeSigningRoot(header1.Header, domain)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	header1.Signature = privKey.Sign(signingRoot[:]).Marshal()[:]

	header2 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: 1,
			Slot:          0,
			ParentRoot:    someRoot2[:],
			StateRoot:     someRoot2[:],
			BodyRoot:      someRoot2[:],
		},
	}
	signingRoot, err = helpers.ComputeSigningRoot(header2.Header, domain)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	header2.Signature = privKey.Sign(signingRoot[:]).Marshal()[:]

	slashing := &ethpb.ProposerSlashing{
		Header_1: header1,
		Header_2: header2,
	}
	val, err := state.ValidatorAtIndex(1)
	if err != nil {
		t.Fatal(err)
	}
	val.PublicKey = privKey.PublicKey().Marshal()[:]
	if err := state.UpdateValidatorAtIndex(1, val); err != nil {
		t.Fatal(err)
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}

	return slashing, state
}

func TestValidateProposerSlashing_ValidSlashing(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	slashing, s := setupValidProposerSlashing(t)

	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	r := &Service{
		p2p:                       p,
		chain:                     &mock.ChainService{State: s},
		initialSync:               &mockSync.Sync{IsSyncing: false},
		seenProposerSlashingCache: c,
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().EncodeGossip(buf, slashing); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(slashing)],
			},
		},
	}

	valid := r.validateProposerSlashing(ctx, "", m) == pubsub.ValidationAccept
	if !valid {
		t.Error("Failed validation")
	}

	if m.ValidatorData == nil {
		t.Error("Decoded message was not set on the message validator data")
	}
}

func TestValidateProposerSlashing_ContextTimeout(t *testing.T) {
	p := p2ptest.NewTestP2P(t)

	slashing, state := setupValidProposerSlashing(t)
	slashing.Header_1.Header.Slot = 100000000
	err := state.SetJustificationBits(bitfield.Bitvector4{0x0F}) // 0b1111
	if err != nil {
		t.Fatal(err)
	}
	err = state.SetPreviousJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 0, Root: []byte{}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	r := &Service{
		p2p:                       p,
		chain:                     &mock.ChainService{State: state},
		initialSync:               &mockSync.Sync{IsSyncing: false},
		seenProposerSlashingCache: c,
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().EncodeGossip(buf, slashing); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(slashing)],
			},
		},
	}
	valid := r.validateProposerSlashing(ctx, "", m) == pubsub.ValidationAccept
	if valid {
		t.Error("slashing from the far distant future should have timed out and returned false")
	}
}

func TestValidateProposerSlashing_Syncing(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	slashing, s := setupValidProposerSlashing(t)

	r := &Service{
		p2p:         p,
		chain:       &mock.ChainService{State: s},
		initialSync: &mockSync.Sync{IsSyncing: true},
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().EncodeGossip(buf, slashing); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(slashing)],
			},
		},
	}
	valid := r.validateProposerSlashing(ctx, "", m) == pubsub.ValidationAccept
	if valid {
		t.Error("Did not fail validation")
	}
}

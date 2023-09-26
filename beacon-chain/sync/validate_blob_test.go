package sync

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	dbtest "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/v4/beacon-chain/sync/initial-sync/testing"
	lruwrpr "github.com/prysmaticlabs/prysm/v4/cache/lru"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestValidateBlob_FromSelf(t *testing.T) {
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	s := &Service{cfg: &config{p2p: p}}
	result, err := s.validateBlob(ctx, s.cfg.p2p.PeerID(), nil)
	require.NoError(t, err)
	require.Equal(t, result, pubsub.ValidationAccept)
}

func TestValidateBlob_InitSync(t *testing.T) {
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	s := &Service{cfg: &config{p2p: p, initialSync: &mockSync.Sync{IsSyncing: true}}}
	result, err := s.validateBlob(ctx, "", nil)
	require.NoError(t, err)
	require.Equal(t, result, pubsub.ValidationIgnore)
}

func TestValidateBlob_InvalidTopic(t *testing.T) {
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	s := &Service{cfg: &config{p2p: p, initialSync: &mockSync.Sync{}}}
	result, err := s.validateBlob(ctx, "", &pubsub.Message{
		Message: &pb.Message{},
	})
	require.ErrorIs(t, errInvalidTopic, err)
	require.Equal(t, result, pubsub.ValidationReject)
}

func TestValidateBlob_InvalidMessageType(t *testing.T) {
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	chainService := &mock.ChainService{Genesis: time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0)}
	s := &Service{cfg: &config{p2p: p, initialSync: &mockSync.Sync{}, clock: startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot)}}

	msg := util.NewBeaconBlock()
	buf := new(bytes.Buffer)
	_, err := p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(msg)]
	digest, err := s.currentForkDigest()
	require.NoError(t, err)
	topic = s.addDigestToTopic(topic, digest)
	result, err := s.validateBlob(ctx, "", &pubsub.Message{
		Message: &pb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		}})
	require.ErrorIs(t, errWrongMessage, err)
	require.Equal(t, result, pubsub.ValidationReject)
}

func TestValidateBlob_InvalidTopicIndex(t *testing.T) {
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	chainService := &mock.ChainService{Genesis: time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0)}
	s := &Service{cfg: &config{p2p: p, initialSync: &mockSync.Sync{}, clock: startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot)}}

	msg := util.NewBlobsidecar()
	buf := new(bytes.Buffer)
	_, err := p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(msg)]
	digest, err := s.currentForkDigest()
	require.NoError(t, err)
	topic = s.addDigestAndIndexToTopic(topic, digest, 1)
	result, err := s.validateBlob(ctx, "", &pubsub.Message{
		Message: &pb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		}})
	require.ErrorContains(t, "/eth2/f5a5fd42/blob_sidecar_1", err)
	require.Equal(t, result, pubsub.ValidationReject)
}

func TestValidateBlob_OlderThanFinalizedEpoch(t *testing.T) {
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	chainService := &mock.ChainService{Genesis: time.Now(), FinalizedCheckPoint: &eth.Checkpoint{Epoch: 1}}
	s := &Service{cfg: &config{
		p2p:         p,
		initialSync: &mockSync.Sync{},
		chain:       chainService,
		clock:       startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot)}}

	b := util.NewBlobsidecar()
	b.Message.Slot = chainService.CurrentSlot() + 1
	buf := new(bytes.Buffer)
	_, err := p.Encoding().EncodeGossip(buf, b)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(b)]
	digest, err := s.currentForkDigest()
	require.NoError(t, err)
	topic = s.addDigestAndIndexToTopic(topic, digest, 0)
	result, err := s.validateBlob(ctx, "", &pubsub.Message{
		Message: &pb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		}})
	require.ErrorContains(t, "finalized slot 32 greater or equal to blob slot 1", err)
	require.Equal(t, result, pubsub.ValidationIgnore)
}

func TestValidateBlob_HigherThanParentSlot(t *testing.T) {
	db := dbtest.SetupDB(t)
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	chainService := &mock.ChainService{Genesis: time.Now(), FinalizedCheckPoint: &eth.Checkpoint{}, DB: db}
	s := &Service{
		cfg: &config{
			p2p:         p,
			initialSync: &mockSync.Sync{},
			chain:       chainService,
			clock:       startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot)}}

	b := util.NewBlobsidecar()
	b.Message.Slot = chainService.CurrentSlot() + 1
	chainService.BlockSlot = chainService.CurrentSlot() + 1
	bb := util.NewBeaconBlock()
	bb.Block.Slot = b.Message.Slot
	signedBb, err := blocks.NewSignedBeaconBlock(bb)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, signedBb))
	r, err := signedBb.Block().HashTreeRoot()
	require.NoError(t, err)

	b.Message.BlockParentRoot = r[:]

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, b)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(b)]
	digest, err := s.currentForkDigest()
	require.NoError(t, err)
	topic = s.addDigestAndIndexToTopic(topic, digest, 0)
	result, err := s.validateBlob(ctx, "", &pubsub.Message{
		Message: &pb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		}})
	require.ErrorContains(t, "parent block slot 1 greater or equal to blob slot 1", err)
	require.Equal(t, result, pubsub.ValidationReject)
}

func TestValidateBlob_InvalidProposerSignature(t *testing.T) {
	db := dbtest.SetupDB(t)
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	chainService := &mock.ChainService{Genesis: time.Now(), FinalizedCheckPoint: &eth.Checkpoint{}, DB: db}
	stateGen := stategen.New(db, doublylinkedtree.New())
	s := &Service{
		cfg: &config{
			p2p:         p,
			initialSync: &mockSync.Sync{},
			chain:       chainService,
			stateGen:    stateGen,
			clock:       startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot)}}

	b := util.NewBlobsidecar()
	b.Message.Slot = chainService.CurrentSlot() + 1
	sk, err := bls.SecretKeyFromBytes(bytesutil.PadTo([]byte("sk"), 32))
	require.NoError(t, err)
	b.Signature = sk.Sign([]byte("data")).Marshal()

	bb := util.NewBeaconBlock()
	signedBb, err := blocks.NewSignedBeaconBlock(bb)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, signedBb))
	r, err := signedBb.Block().HashTreeRoot()
	require.NoError(t, err)

	b.Message.BlockParentRoot = r[:]

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, b)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(b)]
	digest, err := s.currentForkDigest()
	require.NoError(t, err)
	topic = s.addDigestAndIndexToTopic(topic, digest, 0)
	result, err := s.validateBlob(ctx, "", &pubsub.Message{
		Message: &pb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		}})
	require.ErrorIs(t, signing.ErrSigFailedToVerify, err)
	require.Equal(t, result, pubsub.ValidationReject)
}

func TestValidateBlob_AlreadySeenInCache(t *testing.T) {
	db := dbtest.SetupDB(t)
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	chainService := &mock.ChainService{Genesis: time.Now(), FinalizedCheckPoint: &eth.Checkpoint{}, DB: db}
	stateGen := stategen.New(db, doublylinkedtree.New())
	s := &Service{
		seenBlobCache: lruwrpr.New(10),
		cfg: &config{
			p2p:         p,
			initialSync: &mockSync.Sync{},
			chain:       chainService,
			stateGen:    stateGen,
			clock:       startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot)}}

	b := util.NewBlobsidecar()
	b.Message.Slot = chainService.CurrentSlot() + 1
	beaconState, privKeys := util.DeterministicGenesisState(t, 100)

	bb := util.NewBeaconBlock()
	signedBb, err := blocks.NewSignedBeaconBlock(bb)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, signedBb))
	r, err := signedBb.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, r))

	b.Message.BlockParentRoot = r[:]
	b.Signature, err = signing.ComputeDomainAndSign(beaconState, 0, b.Message, params.BeaconConfig().DomainBlobSidecar, privKeys[0])
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, b)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(b)]
	digest, err := s.currentForkDigest()
	require.NoError(t, err)
	topic = s.addDigestAndIndexToTopic(topic, digest, 0)

	s.setSeenBlobIndex(bytesutil.PadTo([]byte{}, 32), 0)
	result, err := s.validateBlob(ctx, "", &pubsub.Message{
		Message: &pb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		}})
	require.NoError(t, err)
	require.Equal(t, result, pubsub.ValidationIgnore)
}

func TestValidateBlob_IncorrectProposerIndex(t *testing.T) {
	db := dbtest.SetupDB(t)
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	chainService := &mock.ChainService{Genesis: time.Now(), FinalizedCheckPoint: &eth.Checkpoint{}, DB: db}
	stateGen := stategen.New(db, doublylinkedtree.New())
	s := &Service{
		seenBlobCache: lruwrpr.New(10),
		cfg: &config{
			p2p:         p,
			initialSync: &mockSync.Sync{},
			chain:       chainService,
			stateGen:    stateGen,
			clock:       startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot)}}

	b := util.NewBlobsidecar()
	b.Message.Slot = chainService.CurrentSlot() + 1
	beaconState, privKeys := util.DeterministicGenesisState(t, 100)

	bb := util.NewBeaconBlock()
	signedBb, err := blocks.NewSignedBeaconBlock(bb)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, signedBb))
	r, err := signedBb.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, r))

	b.Message.BlockParentRoot = r[:]
	b.Signature, err = signing.ComputeDomainAndSign(beaconState, 0, b.Message, params.BeaconConfig().DomainBlobSidecar, privKeys[0])
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, b)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(b)]
	digest, err := s.currentForkDigest()
	require.NoError(t, err)
	topic = s.addDigestAndIndexToTopic(topic, digest, 0)

	result, err := s.validateBlob(ctx, "", &pubsub.Message{
		Message: &pb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		}})
	require.ErrorContains(t, "expected proposer index 21, got 0", err)
	require.Equal(t, result, pubsub.ValidationReject)
}

func TestValidateBlob_EverythingPasses(t *testing.T) {
	db := dbtest.SetupDB(t)
	ctx := context.Background()
	p := p2ptest.NewTestP2P(t)
	chainService := &mock.ChainService{Genesis: time.Now(), FinalizedCheckPoint: &eth.Checkpoint{}, DB: db}
	stateGen := stategen.New(db, doublylinkedtree.New())
	s := &Service{
		seenBlobCache: lruwrpr.New(10),
		cfg: &config{
			p2p:         p,
			initialSync: &mockSync.Sync{},
			chain:       chainService,
			stateGen:    stateGen,
			clock:       startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot)}}

	b := util.NewBlobsidecar()
	b.Message.Slot = chainService.CurrentSlot() + 1
	beaconState, privKeys := util.DeterministicGenesisState(t, 100)

	bb := util.NewBeaconBlock()
	signedBb, err := blocks.NewSignedBeaconBlock(bb)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, signedBb))
	r, err := signedBb.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, r))

	b.Message.BlockParentRoot = r[:]
	b.Message.ProposerIndex = 21
	b.Signature, err = signing.ComputeDomainAndSign(beaconState, 0, b.Message, params.BeaconConfig().DomainBlobSidecar, privKeys[21])
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, b)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(b)]
	digest, err := s.currentForkDigest()
	require.NoError(t, err)
	topic = s.addDigestAndIndexToTopic(topic, digest, 0)

	result, err := s.validateBlob(ctx, "", &pubsub.Message{
		Message: &pb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		}})
	require.NoError(t, err)
	require.Equal(t, result, pubsub.ValidationAccept)
}

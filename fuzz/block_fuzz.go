package fuzz

import (
	"bytes"
	"context"
	"io/ioutil"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/sirupsen/logrus"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	featureconfig.Init(&featureconfig.Flags{SkipBLSVerify: true})

	logrus.SetLevel(logrus.PanicLevel)
	logrus.SetOutput(ioutil.Discard)
}


type fakeChecker struct {}
func (fakeChecker) Syncing() bool {
	return false
}
func (fakeChecker) Status() error {
	return nil
}

func (fakeChecker) Resync() error {
	return nil
}

// BeaconFuzzBlock -- TODO.
func BeaconFuzzBlock(b []byte)  {
	params.UseMainnetConfig()
	input := &InputBlockWithPrestate{}
	if err := input.UnmarshalSSZ(b); err != nil {
		return
	}
	st, err := stateTrie.InitializeFromProtoUnsafe(input.State)
	if err != nil {
		return
	}

	_ = st

	db, err := db.NewDB("/tmp/beacondb", nil)
	if err != nil {
		panic(err)
	}
	defer func () {
		if err := db.ClearDB(); err != nil {
			panic(err)
		}
		if err := db.Close(); err != nil {
			panic(err)
		}
	}()

	p2p := p2pt.NewFuzzTestP2P()

	s := sync.NewRegularSyncFuzz(&sync.Config{
		DB:                  db,
		P2P:                 p2p,
		Chain:               nil,
		InitialSync:         fakeChecker{},
		StateNotifier:       nil,
		BlockNotifier:       nil,
		AttestationNotifier: nil,
		AttPool:             nil,
		ExitPool:            nil,
		SlashingPool:        nil,
		StateSummaryCache:   nil,
		StateGen:            nil,
	})

	buf := new(bytes.Buffer)
	_, err = p2p.Encoding().EncodeGossip(buf, input.Block)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	pid := peer.ID("fuzz")
	msg := &pubsub.Message{
		Message: &pubsub_pb.Message{
			Data:    buf.Bytes(),
			TopicIDs: []string{},
		},
	}
	
	if res := s.FuzzValidateBeaconBlockPubSub(ctx, pid, msg); res != pubsub.ValidationAccept {
		return
	}
	
	if err := s.FuzzBeaconBlockSubscriber(ctx, msg); err != nil {
		_ = err
	}
}

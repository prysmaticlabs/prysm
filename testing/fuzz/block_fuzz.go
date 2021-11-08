// +build libfuzzer

package fuzz

import (
	"bytes"
	"context"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	beaconkv "github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	powt "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/rand"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/sirupsen/logrus"
)

const topic = p2p.BlockSubnetTopicFormat

var db1 db.Database
var dbPath = path.Join(os.TempDir(), "fuzz_beacondb", randomHex(6))

func randomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.NewGenerator().Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func init() {
	logrus.SetLevel(logrus.PanicLevel)
	logrus.SetOutput(ioutil.Discard)

	var err error

	db1, err = db.NewDB(context.Background(), dbPath, &beaconkv.Config{})
	if err != nil {
		panic(err)
	}
}

func setupDB() {
	if err := db1.ClearDB(); err != nil {
		_ = err
	}

	ctx := context.Background()
	s, err := util.NewBeaconState()
	if err != nil {
		panic(err)
	}
	b := util.NewBeaconBlock()
	if err := db1.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b)); err != nil {
		panic(err)
	}
	br, err := b.HashTreeRoot()
	if err != nil {
		panic(err)
	}
	if err := db1.SaveState(ctx, s, br); err != nil {
		panic(err)
	}
	if err := db1.SaveGenesisBlockRoot(ctx, br); err != nil {
		panic(err)
	}
}

type fakeChecker struct{}

func (fakeChecker) Syncing() bool {
	return false
}
func (fakeChecker) Initialized() bool {
	return false
}
func (fakeChecker) Status() error {
	return nil
}
func (fakeChecker) Resync() error {
	return nil
}
func (fakeChecker) Synced() bool {
	return false
}

// FuzzBlock wraps BeaconFuzzBlock in a go-fuzz compatible interface
func FuzzBlock(b []byte) int {
	BeaconFuzzBlock(b)
	return 0
}

// BeaconFuzzBlock runs full processing of beacon block against a given state.
func BeaconFuzzBlock(b []byte) {
	params.UseMainnetConfig()
	input := &InputBlockWithPrestate{}
	if err := input.UnmarshalSSZ(b); err != nil {
		return
	}
	st, err := v1.InitializeFromProtoUnsafe(input.State)
	if err != nil {
		return
	}

	setupDB()

	p2p := p2pt.NewFuzzTestP2P()
	sgen := stategen.New(db1)
	sn := &testing.MockStateNotifier{}
	bn := &testing.MockBlockNotifier{}
	an := &testing.MockOperationNotifier{}
	ap := attestations.NewPool()
	ep := voluntaryexits.NewPool()
	sp := slashings.NewPool()
	ops, err := attestations.NewService(context.Background(), &attestations.Config{Pool: ap})
	if err != nil {
		panic(err)
	}

	chain, err := blockchain.NewService(
		context.Background(),
		blockchain.WithChainStartFetcher(powt.NewPOWChain()),
		blockchain.WithDatabase(db1),
		blockchain.WithAttestationPool(ap),
		blockchain.WithExitPool(ep),
		blockchain.WithSlashingPool(sp),
		blockchain.WithP2PBroadcaster(p2p),
		blockchain.WithStateNotifier(sn),
		blockchain.WithForkChoiceStore(protoarray.New(0, 0, [32]byte{})),
		blockchain.WithAttestationService(ops),
		blockchain.WithStateGen(sgen),
	)
	if err != nil {
		panic(err)
	}
	chain.Start()

	s := sync.NewRegularSyncFuzz(&sync.Config{
		DB:                db1,
		P2P:               p2p,
		Chain:             chain,
		InitialSync:       fakeChecker{},
		StateNotifier:     sn,
		BlockNotifier:     bn,
		OperationNotifier: an,
		AttPool:           ap,
		ExitPool:          ep,
		SlashingPool:      sp,
		StateGen:          sgen,
	})

	s.InitCaches()

	buf := new(bytes.Buffer)
	_, err = p2p.Encoding().EncodeGossip(buf, input.Block)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	pid := peer.ID("fuzz")
	msg := &pubsub.Message{
		Message: &pubsub_pb.Message{
			Data: buf.Bytes(),
			Topic: func() *string {
				tpc := topic
				return &tpc
			}(),
		},
	}

	if res := s.FuzzValidateBeaconBlockPubSub(ctx, pid, msg); res != pubsub.ValidationAccept {
		return
	}

	if err := s.FuzzBeaconBlockSubscriber(ctx, input.Block); err != nil {
		_ = err
	}

	if _, _, err := transition.ProcessBlockNoVerifyAnySig(ctx, st, wrapper.WrappedPhase0SignedBeaconBlock(input.Block)); err != nil {
		_ = err
	}
}

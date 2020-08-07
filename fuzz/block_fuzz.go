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
	"github.com/prysmaticlabs/go-ssz"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/testutil"

	"github.com/sirupsen/logrus"
)

const topic = p2p.BlockSubnetTopicFormat

var db1 db.Database
var ssc *cache.StateSummaryCache
var dbPath = path.Join(os.TempDir(), "fuzz_beacondb", randomHex(6))

func randomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.NewGenerator().Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func init() {
	featureconfig.Init(&featureconfig.Flags{SkipBLSVerify: true})

	logrus.SetLevel(logrus.PanicLevel)
	logrus.SetOutput(ioutil.Discard)

	ssc = cache.NewStateSummaryCache()

	var err error

	db1, err = db.NewDB(dbPath, ssc)
	if err != nil {
		panic(err)
	}
}

func setupDB() {
	if err := db1.ClearDB(); err != nil {
		_ = err
	}

	ctx := context.Background()
	s := testutil.NewBeaconState()
	b := testutil.NewBeaconBlock()
	if err := db1.SaveBlock(ctx, b); err != nil {
		panic(err)
	}
	br, err := ssz.HashTreeRoot(b)
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
func (fakeChecker) Status() error {
	return nil
}

func (fakeChecker) Resync() error {
	return nil
}

// BeaconFuzzBlock -- TODO.
func BeaconFuzzBlock(b []byte) {
	params.UseMainnetConfig()
	input := &InputBlockWithPrestate{}
	if err := input.UnmarshalSSZ(b); err != nil {
		return
	}
	st, err := stateTrie.InitializeFromProtoUnsafe(input.State)
	if err != nil {
		return
	}

	setupDB()

	p2p := p2pt.NewFuzzTestP2P()
	sgen := stategen.New(db1, ssc)
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

	chain, err := blockchain.NewService(context.Background(), &blockchain.Config{
		ChainStartFetcher: nil,
		BeaconDB:          db1,
		DepositCache:      nil,
		AttPool:           ap,
		ExitPool:          ep,
		SlashingPool:      sp,
		P2p:               p2p,
		StateNotifier:     sn,
		ForkChoiceStore:   protoarray.New(0, 0, [32]byte{}),
		OpsService:        ops,
		StateGen:          sgen,
	})
	if err != nil {
		panic(err)
	}
	chain.Start()
	//defer func() {
	//	if err := chain.Stop(); err != nil {
	//		_ = err
	//	}
	//}()

	s := sync.NewRegularSyncFuzz(&sync.Config{
		DB:  db1,
		P2P: p2p,
		Chain: chain,
		//Chain: &testing.ChainService{
		//	FinalizedCheckPoint:         st.FinalizedCheckpoint(),
		//	CurrentJustifiedCheckPoint:  st.CurrentJustifiedCheckpoint(),
		//	PreviousJustifiedCheckPoint: st.PreviousJustifiedCheckpoint(),
		//	State:                       st,
		//	Genesis:                     st.GenesisUnixTime(),
		//	Fork:                        st.Fork(),
		//	ETH1Data:                    st.Eth1Data(),
		//},
		InitialSync:         fakeChecker{},
		StateNotifier:       sn,
		BlockNotifier:       bn,
		AttestationNotifier: an,
		AttPool:             ap,
		ExitPool:            ep,
		SlashingPool:        sp,
		StateSummaryCache:   ssc,
		StateGen:            sgen,
	})

	if err := s.InitCaches(); err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	_, err = p2p.Encoding().EncodeGossip(buf, input.Block)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	pid := peer.ID("fuzz")
	msg := &pubsub.Message{
		Message: &pubsub_pb.Message{
			Data:     buf.Bytes(),
			TopicIDs: []string{topic},
		},
	}

	if res := s.FuzzValidateBeaconBlockPubSub(ctx, pid, msg); res != pubsub.ValidationAccept {
		return
	}

	if err := s.FuzzBeaconBlockSubscriber(ctx, msg); err != nil {
		_ = err
	}

	if _, err := state.ProcessBlock(ctx, st, input.Block); err != nil {
		_ = err
	}
}

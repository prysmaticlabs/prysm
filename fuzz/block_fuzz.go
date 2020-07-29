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
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
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

	if err := db1.ClearDB(); err != nil {
		_ = err
	}

	p2p := p2pt.NewFuzzTestP2P()

	s := sync.NewRegularSyncFuzz(&sync.Config{
		DB:  db1,
		P2P: p2p,
		Chain: &testing.ChainService{
			FinalizedCheckPoint:         st.FinalizedCheckpoint(),
			CurrentJustifiedCheckPoint:  st.CurrentJustifiedCheckpoint(),
			PreviousJustifiedCheckPoint: st.PreviousJustifiedCheckpoint(),
			State:                       st,
			Genesis:                     st.GenesisUnixTime(),
			Fork:                        st.Fork(),
			ETH1Data:                    st.Eth1Data(),
		},
		InitialSync:         fakeChecker{},
		StateNotifier:       &testing.MockStateNotifier{},
		BlockNotifier:       &testing.MockBlockNotifier{},
		AttestationNotifier: &testing.MockOperationNotifier{},
		AttPool:             nil,
		ExitPool:            nil,
		SlashingPool:        nil,
		StateSummaryCache:   ssc,
		StateGen:            stategen.New(db1, ssc),
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

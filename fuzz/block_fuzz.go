// +build libfuzzer

package fuzz

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	beaconkv "github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/testutil"
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
	s, err := testutil.NewBeaconState()
	if err != nil {
		panic(err)
	}
	b := testutil.NewBeaconBlock()
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

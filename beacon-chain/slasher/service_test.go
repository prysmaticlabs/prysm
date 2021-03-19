package slasher

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ = SlashingChecker(&Service{})

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)

	m.Run()
}

func TestService_StartStop_ChainStartEvent(t *testing.T) {
	beaconDB := dbtest.SetupDB(t)
	hook := logTest.NewGlobal()
	srv, err := New(context.Background(), &ServiceConfig{
		IndexedAttestationsFeed: new(event.Feed),
		BeaconBlockHeadersFeed:  new(event.Feed),
		StateNotifier:           &mock.MockStateNotifier{},
		Database:                beaconDB,
	})
	require.NoError(t, err)
	go srv.Start()
	time.Sleep(time.Millisecond * 100)
	srv.serviceCfg.StateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.ChainStarted,
		Data: &statefeed.ChainStartedData{StartTime: time.Now()},
	})
	time.Sleep(time.Millisecond * 100)
	srv.slotTicker = &slotutil.SlotTicker{}
	require.NoError(t, srv.Stop())
	require.NoError(t, srv.Status())
	require.LogsContain(t, hook, "received chain start event")
}

func TestService_StartStop_ChainAlreadyInitialized(t *testing.T) {
	beaconDB := dbtest.SetupDB(t)
	hook := logTest.NewGlobal()
	srv, err := New(context.Background(), &ServiceConfig{
		IndexedAttestationsFeed: new(event.Feed),
		BeaconBlockHeadersFeed:  new(event.Feed),
		StateNotifier:           &mock.MockStateNotifier{},
		Database:                beaconDB,
	})
	require.NoError(t, err)
	go srv.Start()
	time.Sleep(time.Millisecond * 100)
	srv.serviceCfg.StateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Initialized,
		Data: &statefeed.InitializedData{StartTime: time.Now()},
	})
	time.Sleep(time.Millisecond * 100)
	srv.slotTicker = &slotutil.SlotTicker{}
	require.NoError(t, srv.Stop())
	require.NoError(t, srv.Status())
	require.LogsContain(t, hook, "chain already initialized")
}

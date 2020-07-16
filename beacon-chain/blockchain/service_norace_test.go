package blockchain

import (
	"context"
	"io/ioutil"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func TestChainService_SaveHead_DataRace(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	s := &Service{
		beaconDB: db,
	}
	go func() {
		require.NoError(t, s.saveHead(context.Background(), [32]byte{}))
	}()
	require.NoError(t, s.saveHead(context.Background(), [32]byte{}))
}

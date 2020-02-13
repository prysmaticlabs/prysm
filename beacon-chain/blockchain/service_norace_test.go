package blockchain

import (
	"context"
	"io/ioutil"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func TestChainService_SaveHead_DataRace(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	s := &Service{
		beaconDB:       db,
		canonicalRoots: make(map[[40]byte]bool),
	}
	go func() {
		s.saveHead(
			context.Background(),
			[32]byte{},
		)
	}()
	s.saveHead(
		context.Background(),
		[32]byte{},
	)
}

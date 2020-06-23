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
	db, _ := testDB.SetupDB(t)
	s := &Service{
		beaconDB: db,
	}
	go func() {
		if err := s.saveHead(context.Background(), [32]byte{}); err != nil {
			t.Fatal(err)
		}
	}()
	if err := s.saveHead(context.Background(), [32]byte{}); err != nil {
		t.Fatal(err)
	}
}

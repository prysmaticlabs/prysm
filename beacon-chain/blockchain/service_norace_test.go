package blockchain

import (
	"context"
	"io/ioutil"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
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
		canonicalRoots: make(map[uint64][]byte),
	}
	go func() {
		s.saveHead(
			context.Background(),
			&ethpb.BeaconBlock{Slot: 777},
			[32]byte{},
		)
	}()
	s.saveHead(
		context.Background(),
		&ethpb.BeaconBlock{Slot: 888},
		[32]byte{},
	)
}

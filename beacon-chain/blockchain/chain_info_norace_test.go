package blockchain

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestHeadSlot_DataRace(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	s := &Service{
		beaconDB: db,
	}
	go func() {
		require.NoError(t, s.saveHead(context.Background(), [32]byte{}))
	}()
	s.HeadSlot()
}

func TestHeadRoot_DataRace(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	s := &Service{
		beaconDB: db,
		head:     &head{root: [32]byte{'A'}},
		stateGen: stategen.New(db, sc),
	}
	go func() {
		require.NoError(t, s.saveHead(context.Background(), [32]byte{}))
	}()
	_, err := s.HeadRoot(context.Background())
	require.NoError(t, err)
}

func TestHeadBlock_DataRace(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	s := &Service{
		beaconDB: db,
		head:     &head{block: &ethpb.SignedBeaconBlock{}},
		stateGen: stategen.New(db, sc),
	}
	go func() {
		require.NoError(t, s.saveHead(context.Background(), [32]byte{}))
	}()
	_, err := s.HeadBlock(context.Background())
	require.NoError(t, err)
}

func TestHeadState_DataRace(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	s := &Service{
		beaconDB: db,
		stateGen: stategen.New(db, sc),
	}
	go func() {
		require.NoError(t, s.saveHead(context.Background(), [32]byte{}))
	}()
	_, err := s.HeadState(context.Background())
	require.NoError(t, err)
}

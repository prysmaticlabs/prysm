package slasher

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func Test_processQueuedBlocks_DetectsDoubleProposals(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbtest.SetupDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		serviceCfg: &ServiceConfig{
			Database: beaconDB,
		},
		params:            DefaultParams(),
		beaconBlocksQueue: make([]*slashertypes.CompactBeaconBlock, 0),
	}
	currentSlotChan := make(chan types.Slot)
	exitChan := make(chan struct{})
	go func() {
		s.processQueuedBlocks(ctx, currentSlotChan)
		exitChan <- struct{}{}
	}()
	s.beaconBlocksQueue = []*slashertypes.CompactBeaconBlock{
		{
			Slot:          4,
			ProposerIndex: 1,
			SigningRoot:   [32]byte{1},
		},
		{
			Slot:          4,
			ProposerIndex: 1,
			SigningRoot:   [32]byte{1},
		},
		{
			Slot:          4,
			ProposerIndex: 1,
			SigningRoot:   [32]byte{1},
		},
		{
			Slot:          4,
			ProposerIndex: 1,
			SigningRoot:   [32]byte{2},
		},
	}
	currentSlot := types.Slot(4)
	currentSlotChan <- currentSlot
	cancel()
	<-exitChan
	require.LogsContain(t, hook, "Proposer double proposal slashing")
}

func Test_processQueuedBlocks_NotSlashable(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbtest.SetupDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		serviceCfg: &ServiceConfig{
			Database: beaconDB,
		},
		params:            DefaultParams(),
		beaconBlocksQueue: make([]*slashertypes.CompactBeaconBlock, 0),
	}
	currentSlotChan := make(chan types.Slot)
	exitChan := make(chan struct{})
	go func() {
		s.processQueuedBlocks(ctx, currentSlotChan)
		exitChan <- struct{}{}
	}()
	s.beaconBlocksQueue = []*slashertypes.CompactBeaconBlock{
		{
			Slot:          4,
			ProposerIndex: 1,
			SigningRoot:   [32]byte{1},
		},
		{
			Slot:          4,
			ProposerIndex: 1,
			SigningRoot:   [32]byte{1},
		},
	}
	currentSlot := types.Slot(4)
	currentSlotChan <- currentSlot
	cancel()
	<-exitChan
	require.LogsDoNotContain(t, hook, "Proposer double proposal slashing")
}

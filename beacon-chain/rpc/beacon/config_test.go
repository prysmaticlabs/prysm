package beacon

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	statetrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestServer_GetBeaconConfig(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	bs := &Server{}
	res, err := bs.GetBeaconConfig(ctx, &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	conf := params.BeaconConfig()
	numFields := reflect.TypeOf(conf).Elem().NumField()

	// Check if the result has the same number of items as our config struct.
	if len(res.Config) != numFields {
		t.Errorf("Expected %d items in config result, got %d", numFields, len(res.Config))
	}
	want := fmt.Sprintf("%d", conf.Eth1FollowDistance)

	// Check that an element is properly populated from the config.
	if res.Config["Eth1FollowDistance"] != want {
		t.Errorf("Wanted %s for eth1 follow distance, received %s", want, res.Config["Eth1FollowDistance"])
	}
}

func TestServer_GetChainInfo(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()
	addr := common.Address{1, 2, 3}
	if err := db.SaveDepositContractAddress(ctx, addr); err != nil {
		t.Fatal(err)
	}

	finalizedBlock := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: []byte{'A'}}}
	db.SaveBlock(context.Background(), finalizedBlock)
	fRoot, _ := ssz.HashTreeRoot(finalizedBlock.Block)
	justifiedBlock := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2, ParentRoot: []byte{'B'}}}
	db.SaveBlock(context.Background(), justifiedBlock)
	jRoot, _ := ssz.HashTreeRoot(justifiedBlock.Block)
	prevJustifiedBlock := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 3, ParentRoot: []byte{'C'}}}
	db.SaveBlock(context.Background(), prevJustifiedBlock)
	pjRoot, _ := ssz.HashTreeRoot(prevJustifiedBlock.Block)

	st, err := statetrie.InitializeFromProto(&pbp2p.BeaconState{
		Slot:                        1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 3, Root: pjRoot[:]},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Epoch: 2, Root: jRoot[:]},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Epoch: 1, Root: fRoot[:]},
		Fork: &pbp2p.Fork{
			Epoch:           5,
			PreviousVersion: []byte{0x04, 0x04, 0x04, 0x04},
			CurrentVersion:  []byte{0x05, 0x05, 0x05, 0x05},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: st.PreviousJustifiedCheckpoint().Epoch*params.BeaconConfig().SlotsPerEpoch + 1}}
	chain := &mock.ChainService{
		Genesis:                     time.Unix(0, 0),
		Block:                       b,
		State:                       st,
		FinalizedCheckPoint:         st.FinalizedCheckpoint(),
		CurrentJustifiedCheckPoint:  st.CurrentJustifiedCheckpoint(),
		PreviousJustifiedCheckPoint: st.PreviousJustifiedCheckpoint(),
	}
	bs := &Server{
		BeaconDB:           db,
		HeadFetcher:        chain,
		GenesisTimeFetcher: chain,
	}

	chainInfo, err := bs.GetChainInfo(ctx, &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}

	if chainInfo.GenesisTime.Seconds != bs.GenesisTimeFetcher.GenesisTime().Unix() {
		t.Errorf("GenesisTime: expected %v, received %v", bs.GenesisTimeFetcher.GenesisTime(), chainInfo.GenesisTime)
	}

	if !bytes.Equal(chainInfo.DepositContractAddress, addr.Bytes()) {
		t.Errorf("DepositContractAddress: expected %x, received %x", addr.Bytes(), chainInfo.DepositContractAddress)
	}

	if chainInfo.SecondsPerSlot != params.BeaconConfig().SecondsPerSlot {
		t.Errorf("SecondsPerSlot: expected %d, received %d", params.BeaconConfig().SecondsPerSlot, chainInfo.SecondsPerSlot)
	}

	if chainInfo.SlotsPerEpoch != params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("SlotsPerEpoch: expected %d, received %d", params.BeaconConfig().SlotsPerEpoch, chainInfo.SlotsPerEpoch)
	}

	if chainInfo.MaxSeedLookahead != params.BeaconConfig().MaxSeedLookahead {
		t.Errorf("MaxSeedLookahead: expected %d, received %d", params.BeaconConfig().MaxSeedLookahead, chainInfo.MaxSeedLookahead)
	}

	if chainInfo.MinValidatorWithdrawabilityDelay != params.BeaconConfig().MinValidatorWithdrawabilityDelay {
		t.Errorf("MinValidatorWithdrawabilityDelay: expected %d, received %d", params.BeaconConfig().MinValidatorWithdrawabilityDelay, chainInfo.MinValidatorWithdrawabilityDelay)
	}

	if chainInfo.PersistentCommitteePeriod != params.BeaconConfig().PersistentCommitteePeriod {
		t.Errorf("PersistentCommitteePeriod: expected %d, received %d", params.BeaconConfig().PersistentCommitteePeriod, chainInfo.PersistentCommitteePeriod)
	}

	if chainInfo.MinEpochsToInactivityPenalty != params.BeaconConfig().MinEpochsToInactivityPenalty {
		t.Errorf("MinEpochsToInactivityPenalty: expected %d, received %d", params.BeaconConfig().MinEpochsToInactivityPenalty, chainInfo.MinEpochsToInactivityPenalty)
	}

	if chainInfo.Eth1FollowDistance != params.BeaconConfig().Eth1FollowDistance {
		t.Errorf("Eth1FollowDistance: expected %d, received %d", params.BeaconConfig().Eth1FollowDistance, chainInfo.Eth1FollowDistance)
	}

	if chainInfo.FarFutureEpoch != params.BeaconConfig().FarFutureEpoch {
		t.Errorf("FarFutureEpoch: expected %d, received %d", params.BeaconConfig().FarFutureEpoch, chainInfo.FarFutureEpoch)
	}

	if !bytes.Equal(chainInfo.GenesisForkVersion, params.BeaconConfig().GenesisForkVersion) {
		t.Errorf("GenesisForkVersion: expected %x, received %x", params.BeaconConfig().GenesisForkVersion, chainInfo.GenesisForkVersion)
	}

	if !bytes.Equal(chainInfo.GenesisValidatorsRoot, st.GenesisValidatorRoot()) {
		t.Errorf("GenesisValidatorsRoot: expected %x, received %x", st.GenesisValidatorRoot(), chainInfo.GenesisValidatorsRoot)
	}

	if chainInfo.MinimumDepositAmount != params.BeaconConfig().MinDepositAmount {
		t.Errorf("MinimumDepositAmount: expected %v, received %v", params.BeaconConfig().MinDepositAmount, chainInfo.MinimumDepositAmount)
	}

	if chainInfo.MaximumEffectiveBalance != params.BeaconConfig().MaxEffectiveBalance {
		t.Errorf("MaximumEffectiveBalance: expected %v, received %v", params.BeaconConfig().MaxEffectiveBalance, chainInfo.MaximumEffectiveBalance)
	}

	if chainInfo.EffectiveBalanceIncrement != params.BeaconConfig().EffectiveBalanceIncrement {
		t.Errorf("EffectiveBalanceIncrement: expected %v, received %v", params.BeaconConfig().EffectiveBalanceIncrement, chainInfo.EffectiveBalanceIncrement)
	}

	if chainInfo.EjectionBalance != params.BeaconConfig().EjectionBalance {
		t.Errorf("EjectionBalance: expected %v, received %v", params.BeaconConfig().EjectionBalance, chainInfo.EjectionBalance)
	}

	if !bytes.Equal(chainInfo.BlsWithdrawalPrefix, []byte{params.BeaconConfig().BLSWithdrawalPrefixByte}) {
		t.Errorf("BlsWithdrawalPrefix: expected %x, received %x", []byte{params.BeaconConfig().BLSWithdrawalPrefixByte}, chainInfo.BlsWithdrawalPrefix)
	}

	if !bytes.Equal(chainInfo.PreviousForkVersion, st.Fork().PreviousVersion) {
		t.Errorf("PreviousForkVersion: expected %x, received %x", st.Fork().PreviousVersion, chainInfo.PreviousForkVersion)
	}

	if !bytes.Equal(chainInfo.CurrentForkVersion, st.Fork().CurrentVersion) {
		t.Errorf("CurrentForkVersion: expected %x, received %x", st.Fork().CurrentVersion, chainInfo.CurrentForkVersion)
	}

	if chainInfo.CurrentForkEpoch != st.Fork().Epoch {
		t.Errorf("CurrentForkEpoch: expected %v, received %v", st.Fork().Epoch, chainInfo.CurrentForkEpoch)
	}

	if !bytes.Equal(chainInfo.NextForkVersion, params.BeaconConfig().NextForkVersion) {
		t.Errorf("NextForkVersion: expected %x, received %x", params.BeaconConfig().NextForkVersion, chainInfo.NextForkVersion)
	}

	if chainInfo.NextForkEpoch != params.BeaconConfig().NextForkEpoch {
		t.Errorf("NextForkEpoch: expected %v, received %v", params.BeaconConfig().NextForkEpoch, chainInfo.NextForkEpoch)
	}

	if chainInfo.CurrentEpoch != helpers.CurrentEpoch(st) {
		t.Errorf("CurrentEpoch: expected %v, received %v", helpers.CurrentEpoch(st), chainInfo.CurrentEpoch)
	}

	if len(chainInfo.ForkVersionSchedule) != 0 {
		t.Errorf("ForkVersionSchedule: expected length %v, received %v", 0, len(chainInfo.ForkVersionSchedule))
	}
}

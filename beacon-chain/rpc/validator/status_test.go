package validator

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func TestValidatorStatus_DepositedEth1(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()
	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatalf("Could not generate deposits and keys: %v", err)
	}
	deposit := deposits[0]
	pubKey1 := deposit.Data.PublicKey
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not setup deposit trie: %v", err)
	}
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.Root())
	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	stateObj, err := stateTrie.InitializeFromProtoUnsafe(&pbp2p.BeaconState{})
	if err != nil {
		t.Fatal(err)
	}
	vs := &Server{
		BeaconDB:       db,
		DepositFetcher: depositCache,
		BlockFetcher:   p,
		HeadFetcher: &mockChain.ChainService{
			State: stateObj,
		},
		Eth1InfoFetcher: p,
	}
	req := &ethpb.ValidatorStatusRequest{
		PublicKey: pubKey1,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != ethpb.ValidatorStatus_DEPOSITED {
		t.Errorf("Wanted %v, got %v", ethpb.ValidatorStatus_DEPOSITED, resp.Status)
	}
}

func TestValidatorStatus_Deposited(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	pubKey1 := pubKey(1)
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey1,
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
	}
	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not setup deposit trie: %v", err)
	}
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.Root())
	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	stateObj, err := stateTrie.InitializeFromProtoUnsafe(&pbp2p.BeaconState{
		Validators: []*ethpb.Validator{
			{
				PublicKey:                  pubKey1,
				ActivationEligibilityEpoch: 1,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	vs := &Server{
		BeaconDB:       db,
		DepositFetcher: depositCache,
		BlockFetcher:   p,
		HeadFetcher: &mockChain.ChainService{
			State: stateObj,
		},
		Eth1InfoFetcher: p,
	}
	req := &ethpb.ValidatorStatusRequest{
		PublicKey: pubKey1,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != ethpb.ValidatorStatus_DEPOSITED {
		t.Errorf("Wanted %v, got %v", ethpb.ValidatorStatus_DEPOSITED, resp.Status)
	}
}

func TestValidatorStatus_Pending(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	pubKey := pubKey(1)
	block := testutil.NewBeaconBlock()
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}
	// Pending active because activation epoch is still defaulted at far future slot.
	state := testutil.NewBeaconState()
	if err := state.SetSlot(5000); err != nil {
		t.Fatal(err)
	}
	if err := state.SetValidators([]*ethpb.Validator{
		{
			ActivationEpoch:   params.BeaconConfig().FarFutureEpoch,
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey:         pubKey,
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, state, genesisRoot); err != nil {
		t.Fatalf("could not save state: %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, genesisRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not setup deposit trie: %v", err)
	}
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.Root())

	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	vs := &Server{
		BeaconDB:          db,
		ChainStartFetcher: p,
		BlockFetcher:      p,
		Eth1InfoFetcher:   p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mockChain.ChainService{State: state, Root: genesisRoot[:]},
	}
	req := &ethpb.ValidatorStatusRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != ethpb.ValidatorStatus_PENDING {
		t.Errorf("Wanted %v, got %v", ethpb.ValidatorStatus_PENDING, resp.Status)
	}
}

func TestValidatorStatus_Active(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	// This test breaks if it doesnt use mainnet config
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	ctx := context.Background()

	pubKey := pubKey(1)

	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not setup deposit trie: %v", err)
	}
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.Root())

	// Active because activation epoch <= current epoch < exit epoch.
	activeEpoch := helpers.ActivationExitEpoch(0)

	block := testutil.NewBeaconBlock()
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	state := &pbp2p.BeaconState{
		GenesisTime: uint64(time.Unix(0, 0).Unix()),
		Slot:        10000,
		Validators: []*ethpb.Validator{{
			ActivationEpoch:   activeEpoch,
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey:         pubKey},
		}}
	stateObj, err := stateTrie.InitializeFromProtoUnsafe(state)
	if err != nil {
		t.Fatal(err)
	}

	timestamp := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			int(params.BeaconConfig().Eth1FollowDistance): uint64(timestamp),
		},
	}
	vs := &Server{
		BeaconDB:          db,
		ChainStartFetcher: p,
		BlockFetcher:      p,
		Eth1InfoFetcher:   p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mockChain.ChainService{State: stateObj, Root: genesisRoot[:]},
	}
	req := &ethpb.ValidatorStatusRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}

	expected := &ethpb.ValidatorStatusResponse{
		Status:          ethpb.ValidatorStatus_ACTIVE,
		ActivationEpoch: 5,
	}
	if !proto.Equal(resp, expected) {
		t.Errorf("Wanted %v, got %v", expected, resp)
	}
}

func TestValidatorStatus_Exiting(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	pubKey := pubKey(1)

	// Initiated exit because validator exit epoch and withdrawable epoch are not FAR_FUTURE_EPOCH
	slot := uint64(10000)
	epoch := helpers.SlotToEpoch(slot)
	exitEpoch := helpers.ActivationExitEpoch(epoch)
	withdrawableEpoch := exitEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
	block := testutil.NewBeaconBlock()
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	state := &pbp2p.BeaconState{
		Slot: slot,
		Validators: []*ethpb.Validator{{
			PublicKey:         pubKey,
			ActivationEpoch:   0,
			ExitEpoch:         exitEpoch,
			WithdrawableEpoch: withdrawableEpoch},
		}}
	stateObj, err := stateTrie.InitializeFromProtoUnsafe(state)
	if err != nil {
		t.Fatal(err)
	}
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not setup deposit trie: %v", err)
	}
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.Root())
	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	vs := &Server{
		BeaconDB:          db,
		ChainStartFetcher: p,
		BlockFetcher:      p,
		Eth1InfoFetcher:   p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mockChain.ChainService{State: stateObj, Root: genesisRoot[:]},
	}
	req := &ethpb.ValidatorStatusRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != ethpb.ValidatorStatus_EXITING {
		t.Errorf("Wanted %v, got %v", ethpb.ValidatorStatus_EXITING, resp.Status)
	}
}

func TestValidatorStatus_Slashing(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	pubKey := pubKey(1)

	// Exit slashed because slashed is true, exit epoch is =< current epoch and withdrawable epoch > epoch .
	slot := uint64(10000)
	epoch := helpers.SlotToEpoch(slot)
	block := testutil.NewBeaconBlock()
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	state := &pbp2p.BeaconState{
		Slot: slot,
		Validators: []*ethpb.Validator{{
			Slashed:           true,
			PublicKey:         pubKey,
			WithdrawableEpoch: epoch + 1},
		}}
	stateObj, err := stateTrie.InitializeFromProtoUnsafe(state)
	if err != nil {
		t.Fatal(err)
	}
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not setup deposit trie: %v", err)
	}
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.Root())
	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	vs := &Server{
		BeaconDB:          db,
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		DepositFetcher:    depositCache,
		BlockFetcher:      p,
		HeadFetcher:       &mockChain.ChainService{State: stateObj, Root: genesisRoot[:]},
	}
	req := &ethpb.ValidatorStatusRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != ethpb.ValidatorStatus_EXITED {
		t.Errorf("Wanted %v, got %v", ethpb.ValidatorStatus_EXITED, resp.Status)
	}
}

func TestValidatorStatus_Exited(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	pubKey := pubKey(1)

	// Exit because only exit epoch is =< current epoch.
	slot := uint64(10000)
	epoch := helpers.SlotToEpoch(slot)
	block := testutil.NewBeaconBlock()
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	numDeposits := uint64(64)
	beaconState, _ := testutil.DeterministicGenesisState(t, numDeposits)
	if err := db.SaveState(ctx, beaconState, genesisRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, genesisRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}
	state := testutil.NewBeaconState()
	if err := state.SetSlot(slot); err != nil {
		t.Fatal(err)
	}
	if err := state.SetValidators([]*ethpb.Validator{{
		PublicKey:         pubKey,
		WithdrawableEpoch: epoch + 1},
	}); err != nil {
		t.Fatal(err)
	}
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not setup deposit trie: %v", err)
	}
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.Root())
	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	vs := &Server{
		BeaconDB:          db,
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		BlockFetcher:      p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mockChain.ChainService{State: state, Root: genesisRoot[:]},
	}
	req := &ethpb.ValidatorStatusRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != ethpb.ValidatorStatus_EXITED {
		t.Errorf("Wanted %v, got %v", ethpb.ValidatorStatus_EXITED, resp.Status)
	}
}

func TestValidatorStatus_UnknownStatus(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	pubKey := pubKey(1)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	stateObj, err := stateTrie.InitializeFromProtoUnsafe(&pbp2p.BeaconState{
		Slot: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	vs := &Server{
		DepositFetcher:  depositCache,
		Eth1InfoFetcher: &mockPOW.POWChain{},
		HeadFetcher: &mockChain.ChainService{
			State: stateObj,
		},
		BeaconDB: db,
	}
	req := &ethpb.ValidatorStatusRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != ethpb.ValidatorStatus_UNKNOWN_STATUS {
		t.Errorf("Wanted %v, got %v", ethpb.ValidatorStatus_UNKNOWN_STATUS, resp.Status)
	}
}

func TestActivationStatus_OK(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	deposits, _, err := testutil.DeterministicDepositsAndKeys(4)
	pubKeys := [][]byte{deposits[0].Data.PublicKey, deposits[1].Data.PublicKey, deposits[2].Data.PublicKey, deposits[3].Data.PublicKey}
	stateObj, err := stateTrie.InitializeFromProtoUnsafe(&pbp2p.BeaconState{
		Slot: 4000,
		Validators: []*ethpb.Validator{
			{
				ActivationEpoch: 0,
				ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
				PublicKey:       pubKeys[0],
			},
			{
				ActivationEpoch: 0,
				ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
				PublicKey:       pubKeys[1],
			},
			{
				ActivationEligibilityEpoch: 700,
				ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
				PublicKey:                  pubKeys[3],
			},
		},
	})
	block := testutil.NewBeaconBlock()
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}
	dep := deposits[0]
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not setup deposit trie: %v", err)
	}
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, dep, 10 /*blockNum*/, 0, depositTrie.Root())

	dep = deposits[2]
	depositTrie.Insert(dep.Data.Signature, 15)
	depositCache.InsertDeposit(context.Background(), dep, 0, 0, depositTrie.Root())

	vs := &Server{
		BeaconDB:           db,
		Ctx:                context.Background(),
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		DepositFetcher:     depositCache,
		HeadFetcher:        &mockChain.ChainService{State: stateObj, Root: genesisRoot[:]},
	}
	activeExists, response, err := vs.activationStatus(context.Background(), pubKeys)
	if err != nil {
		t.Fatal(err)
	}
	if !activeExists {
		t.Fatal("No activated validator exists when there was supposed to be 2")
	}
	if response[0].Status.Status != ethpb.ValidatorStatus_ACTIVE {
		t.Errorf("Validator with pubkey %#x is not activated and instead has this status: %s",
			response[0].PublicKey, response[0].Status.Status.String())
	}
	if response[0].Index != 0 {
		t.Errorf("Validator with pubkey %#x is expected to have index %d, received %d", response[0].PublicKey, 0, response[0].Index)
	}

	if response[1].Status.Status != ethpb.ValidatorStatus_ACTIVE {
		t.Errorf("Validator with pubkey %#x was activated when not supposed to",
			response[1].PublicKey)
	}
	if response[1].Index != 1 {
		t.Errorf("Validator with pubkey %#x is expected to have index %d, received %d", response[1].PublicKey, 1, response[1].Index)
	}

	if response[2].Status.Status != ethpb.ValidatorStatus_DEPOSITED {
		t.Errorf("Validator with pubkey %#x is not unknown and instead has this status: %s",
			response[2].PublicKey, response[2].Status.Status.String())
	}
	if response[2].Index != params.BeaconConfig().FarFutureEpoch {
		t.Errorf("Validator with pubkey %#x is expected to have index %d, received %d", response[2].PublicKey, params.BeaconConfig().FarFutureEpoch, response[2].Index)
	}

	if response[3].Status.Status != ethpb.ValidatorStatus_DEPOSITED {
		t.Errorf("Validator with pubkey %#x was not deposited and has this status: %s",
			response[3].PublicKey, response[3].Status.Status.String())
	}
	if response[3].Index != 2 {
		t.Errorf("Validator with pubkey %#x is expected to have index %d, received %d", response[3].PublicKey, 2, response[3].Index)
	}
}

func TestValidatorStatus_CorrectActivationQueue(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	pbKey := pubKey(5)
	block := testutil.NewBeaconBlock()
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}
	currentSlot := uint64(5000)
	// Pending active because activation epoch is still defaulted at far future slot.
	validators := []*ethpb.Validator{
		{
			ActivationEpoch:   0,
			PublicKey:         pubKey(0),
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
		},
		{
			ActivationEpoch:   0,
			PublicKey:         pubKey(1),
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
		},
		{
			ActivationEpoch:   0,
			PublicKey:         pubKey(2),
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
		},
		{
			ActivationEpoch:   0,
			PublicKey:         pubKey(3),
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
		},
		{
			ActivationEpoch:   currentSlot/params.BeaconConfig().SlotsPerEpoch + 1,
			PublicKey:         pbKey,
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
		},
		{
			ActivationEpoch:   currentSlot/params.BeaconConfig().SlotsPerEpoch + 4,
			PublicKey:         pubKey(5),
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	state := testutil.NewBeaconState()
	if err := state.SetValidators(validators); err != nil {
		t.Fatal(err)
	}
	if err := state.SetSlot(currentSlot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, state, genesisRoot); err != nil {
		t.Fatalf("could not save state: %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, genesisRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not setup deposit trie: %v", err)
	}
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	for i := 0; i < 6; i++ {
		depData := &ethpb.Deposit_Data{
			PublicKey:             pubKey(uint64(i)),
			Signature:             []byte("hi"),
			WithdrawalCredentials: []byte("hey"),
		}

		deposit := &ethpb.Deposit{
			Data: depData,
		}
		depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.Root())

	}

	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	vs := &Server{
		BeaconDB:          db,
		ChainStartFetcher: p,
		BlockFetcher:      p,
		Eth1InfoFetcher:   p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mockChain.ChainService{State: state, Root: genesisRoot[:]},
	}
	req := &ethpb.ValidatorStatusRequest{
		PublicKey: pbKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != ethpb.ValidatorStatus_PENDING {
		t.Errorf("Wanted %v, got %v", ethpb.ValidatorStatus_PENDING, resp.Status)
	}
	if resp.PositionInActivationQueue != 2 {
		t.Errorf("Expected Position in activation queue of %d but instead got %d", 2, resp.PositionInActivationQueue)
	}
}

func TestDepositBlockSlotAfterGenesisTime(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	pubKey := pubKey(1)

	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not setup deposit trie: %v", err)
	}
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.Root())

	timestamp := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			int(params.BeaconConfig().Eth1FollowDistance): uint64(timestamp),
		},
	}

	block := testutil.NewBeaconBlock()
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	activeEpoch := helpers.ActivationExitEpoch(0)

	state, err := stateTrie.InitializeFromProtoUnsafe(&pbp2p.BeaconState{
		GenesisTime: uint64(time.Unix(0, 0).Unix()),
		Slot:        10000,
		Validators: []*ethpb.Validator{{
			ActivationEpoch: activeEpoch,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			PublicKey:       pubKey},
		}})
	if err != nil {
		t.Fatal(err)
	}

	vs := &Server{
		BeaconDB:          db,
		ChainStartFetcher: p,
		BlockFetcher:      p,
		Eth1InfoFetcher:   p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mockChain.ChainService{State: state, Root: genesisRoot[:]},
	}

	eth1BlockNumBigInt := big.NewInt(1000000)

	resp, err := vs.depositBlockSlot(context.Background(), state, eth1BlockNumBigInt)
	if err != nil {
		t.Fatalf("Could not get the deposit block slot %v", err)
	}

	expected := uint64(69)

	if resp != expected {
		t.Errorf("Wanted %v, got %v", expected, resp)
	}
}

func TestDepositBlockSlotBeforeGenesisTime(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	pubKey := pubKey(1)

	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not setup deposit trie: %v", err)
	}
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.Root())

	timestamp := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			int(params.BeaconConfig().Eth1FollowDistance): uint64(timestamp),
		},
	}

	block := testutil.NewBeaconBlock()
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	activeEpoch := helpers.ActivationExitEpoch(0)

	state, err := stateTrie.InitializeFromProtoUnsafe(&pbp2p.BeaconState{
		GenesisTime: uint64(time.Unix(25000, 0).Unix()),
		Slot:        10000,
		Validators: []*ethpb.Validator{{
			ActivationEpoch: activeEpoch,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			PublicKey:       pubKey},
		}})
	if err != nil {
		t.Fatal(err)
	}

	vs := &Server{
		BeaconDB:          db,
		ChainStartFetcher: p,
		BlockFetcher:      p,
		Eth1InfoFetcher:   p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mockChain.ChainService{State: state, Root: genesisRoot[:]},
	}

	eth1BlockNumBigInt := big.NewInt(1000000)
	resp, err := vs.depositBlockSlot(context.Background(), state, eth1BlockNumBigInt)
	if err != nil {
		t.Fatalf("Could not get the deposit block slot %v", err)
	}

	expected := uint64(0)

	if resp != expected {
		t.Errorf("Wanted %v, got %v", expected, resp)
	}
}

func TestMultipleValidatorStatus_Pubkeys(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	deposits, _, err := testutil.DeterministicDepositsAndKeys(4)
	pubKeys := [][]byte{deposits[0].Data.PublicKey, deposits[1].Data.PublicKey, deposits[2].Data.PublicKey, deposits[3].Data.PublicKey}
	stateObj, err := stateTrie.InitializeFromProtoUnsafe(&pbp2p.BeaconState{
		Slot: 4000,
		Validators: []*ethpb.Validator{
			{
				ActivationEpoch: 0,
				ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
				PublicKey:       pubKeys[0],
			},
			{
				ActivationEpoch: 0,
				ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
				PublicKey:       pubKeys[1],
			},
			{
				ActivationEligibilityEpoch: 700,
				ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
				PublicKey:                  pubKeys[3],
			},
		},
	})
	block := testutil.NewBeaconBlock()
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}
	dep := deposits[0]
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not setup deposit trie: %v", err)
	}
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, dep, 10 /*blockNum*/, 0, depositTrie.Root())
	dep = deposits[2]
	depositTrie.Insert(dep.Data.Signature, 15)
	depositCache.InsertDeposit(context.Background(), dep, 0, 0, depositTrie.Root())

	vs := &Server{
		BeaconDB:           db,
		Ctx:                context.Background(),
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		DepositFetcher:     depositCache,
		HeadFetcher:        &mockChain.ChainService{State: stateObj, Root: genesisRoot[:]},
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
	}

	want := []*ethpb.ValidatorStatusResponse{
		{
			Status: ethpb.ValidatorStatus_ACTIVE,
		},
		{
			Status: ethpb.ValidatorStatus_ACTIVE,
		},
		{
			Status:               ethpb.ValidatorStatus_DEPOSITED,
			DepositInclusionSlot: 69,
			ActivationEpoch:      18446744073709551615,
		},
		{
			Status: ethpb.ValidatorStatus_DEPOSITED,
		},
	}

	req := &ethpb.MultipleValidatorStatusRequest{PublicKeys: pubKeys}
	response, err := vs.MultipleValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(response.PublicKeys) != len(pubKeys) {
		t.Fatalf(
			"Recieved %d public keys, wanted %d public keys", len(response.PublicKeys), len(pubKeys))
	}
	for i, resp := range response.PublicKeys {
		if !bytes.Equal(pubKeys[i], resp) {
			t.Fatalf("Wanted: %v\n Recieved: %v\n", pubKeys[i], resp)
		}
	}
	if len(response.Statuses) != len(pubKeys) {
		t.Fatalf("Recieved %d statuses, wanted %d statuses", len(response.Statuses), len(pubKeys))
	}
	for i, resp := range response.Statuses {
		if !proto.Equal(want[i], resp) {
			t.Fatalf("Wanted %v\n Recieved: %v\n", want[i], resp)
		}
	}
}

func TestMultipleValidatorStatus_Indices(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	slot := uint64(10000)
	epoch := helpers.SlotToEpoch(slot)
	pubKeys := [][]byte{pubKey(1), pubKey(2), pubKey(3), pubKey(4)}
	beaconState := &pbp2p.BeaconState{
		Slot: 4000,
		Validators: []*ethpb.Validator{
			{
				ActivationEpoch: 0,
				ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
				PublicKey:       pubKeys[0],
			},
			{
				ActivationEpoch: 0,
				ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
				PublicKey:       pubKeys[1],
			},
			{
				ActivationEligibilityEpoch: 700,
				ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
				PublicKey:                  pubKeys[2],
			},
			{
				Slashed:   true,
				ExitEpoch: epoch + 1,
				PublicKey: pubKeys[3],
			},
		},
	}
	stateObj, err := stateTrie.InitializeFromProtoUnsafe(beaconState)
	block := testutil.NewBeaconBlock()
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	vs := &Server{
		BeaconDB:           db,
		Ctx:                context.Background(),
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		HeadFetcher:        &mockChain.ChainService{State: stateObj, Root: genesisRoot[:]},
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
	}

	want := []*ethpb.ValidatorStatusResponse{
		{
			Status: ethpb.ValidatorStatus_ACTIVE,
		},
		{
			Status: ethpb.ValidatorStatus_ACTIVE,
		},
		{
			Status: ethpb.ValidatorStatus_DEPOSITED,
		},
		{
			Status: ethpb.ValidatorStatus_SLASHING,
		},
	}

	// Note: Index 4 should be skipped.
	req := &ethpb.MultipleValidatorStatusRequest{Indices: []int64{0, 1, 2, 3, 4}}
	response, err := vs.MultipleValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(response.PublicKeys) != len(beaconState.Validators) {
		t.Fatalf(
			"Recieved %d public keys, wanted %d public keys",
			len(response.PublicKeys), len(beaconState.Validators))
	}
	for i, resp := range response.PublicKeys {
		expected := beaconState.Validators[i].PublicKey
		if !bytes.Equal(expected, resp) {
			t.Fatalf("Wanted: %v\n Recieved: %v\n", expected, resp)
		}
	}
	if len(response.Statuses) != len(beaconState.Validators) {
		t.Fatalf(
			"Recieved %d statuses, wanted %d statuses",
			len(response.Statuses), len(beaconState.Validators))
	}
	for i, resp := range response.Statuses {
		if !proto.Equal(want[i], resp) {
			t.Fatalf("Wanted %v\n Recieved: %v\n", want[i], resp)
		}
	}
}

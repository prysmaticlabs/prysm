package validator

import (
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
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func TestValidatorStatus_DepositedEth1(t *testing.T) {
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
	require.NoError(t, err, "Could not setup deposit trie")
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
	require.NoError(t, err)
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
	require.NoError(t, err, "Could not get validator status")
	assert.Equal(t, ethpb.ValidatorStatus_DEPOSITED, resp.Status)
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
	require.NoError(t, err, "Could not setup deposit trie")
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
	require.NoError(t, err)
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
	require.NoError(t, err, "Could not get validator status")
	assert.Equal(t, ethpb.ValidatorStatus_DEPOSITED, resp.Status)
}

func TestValidatorStatus_Pending(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	pubKey := pubKey(1)
	block := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, block), "Could not save genesis block")
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err, "Could not get signing root")
	// Pending active because activation epoch is still defaulted at far future slot.
	state := testutil.NewBeaconState()
	require.NoError(t, state.SetSlot(5000))
	err = state.SetValidators([]*ethpb.Validator{
		{
			ActivationEpoch:   params.BeaconConfig().FarFutureEpoch,
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey:         pubKey,
		},
	})
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, state, genesisRoot), "Could not save state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, genesisRoot), "Could not save genesis state")

	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")
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
	require.NoError(t, err, "Could not get validator status")
	assert.Equal(t, ethpb.ValidatorStatus_PENDING, resp.Status)
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
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.Root())

	// Active because activation epoch <= current epoch < exit epoch.
	activeEpoch := helpers.ActivationExitEpoch(0)

	block := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, block), "Could not save genesis block")
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err, "Could not get signing root")

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
	require.NoError(t, err)

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
	require.NoError(t, err, "Could not get validator status")

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
	require.NoError(t, db.SaveBlock(ctx, block), "Could not save genesis block")
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err, "Could not get signing root")

	state := &pbp2p.BeaconState{
		Slot: slot,
		Validators: []*ethpb.Validator{{
			PublicKey:         pubKey,
			ActivationEpoch:   0,
			ExitEpoch:         exitEpoch,
			WithdrawableEpoch: withdrawableEpoch},
		}}
	stateObj, err := stateTrie.InitializeFromProtoUnsafe(state)
	require.NoError(t, err)
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")
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
	require.NoError(t, err, "Could not get validator status")
	assert.Equal(t, ethpb.ValidatorStatus_EXITING, resp.Status)
}

func TestValidatorStatus_Slashing(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	pubKey := pubKey(1)

	// Exit slashed because slashed is true, exit epoch is =< current epoch and withdrawable epoch > epoch .
	slot := uint64(10000)
	epoch := helpers.SlotToEpoch(slot)
	block := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, block), "Could not save genesis block")
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err, "Could not get signing root")

	state := &pbp2p.BeaconState{
		Slot: slot,
		Validators: []*ethpb.Validator{{
			Slashed:           true,
			PublicKey:         pubKey,
			WithdrawableEpoch: epoch + 1},
		}}
	stateObj, err := stateTrie.InitializeFromProtoUnsafe(state)
	require.NoError(t, err)
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")
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
	require.NoError(t, err, "Could not get validator status")
	assert.Equal(t, ethpb.ValidatorStatus_EXITED, resp.Status)
}

func TestValidatorStatus_Exited(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	pubKey := pubKey(1)

	// Exit because only exit epoch is =< current epoch.
	slot := uint64(10000)
	epoch := helpers.SlotToEpoch(slot)
	block := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, block), "Could not save genesis block")
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err, "Could not get signing root")
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	numDeposits := uint64(64)
	beaconState, _ := testutil.DeterministicGenesisState(t, numDeposits)
	require.NoError(t, db.SaveState(ctx, beaconState, genesisRoot))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, genesisRoot), "Could not save genesis state")
	state := testutil.NewBeaconState()
	require.NoError(t, state.SetSlot(slot))
	err = state.SetValidators([]*ethpb.Validator{{
		PublicKey:         pubKey,
		WithdrawableEpoch: epoch + 1},
	})
	require.NoError(t, err)
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")
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
	require.NoError(t, err, "Could not get validator status")
	assert.Equal(t, ethpb.ValidatorStatus_EXITED, resp.Status)
}

func TestValidatorStatus_UnknownStatus(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	pubKey := pubKey(1)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	stateObj, err := stateTrie.InitializeFromProtoUnsafe(&pbp2p.BeaconState{
		Slot: 0,
	})
	require.NoError(t, err)
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
	require.NoError(t, err, "Could not get validator status")
	assert.Equal(t, ethpb.ValidatorStatus_UNKNOWN_STATUS, resp.Status)
}

func TestActivationStatus_OK(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	pubKeys := [][]byte{pubKey(1), pubKey(2), pubKey(3), pubKey(4)}
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
	require.NoError(t, err, "Could not get signing root")
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey(1),
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
		Amount:                10,
	}

	dep := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, dep, 10 /*blockNum*/, 0, depositTrie.Root())
	depData = &ethpb.Deposit_Data{
		PublicKey:             pubKey(3),
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
		Amount:                10,
	}

	dep = &ethpb.Deposit{
		Data: depData,
	}
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
	require.NoError(t, err)
	require.Equal(t, true, activeExists, "No activated validator exists when there was supposed to be 2")
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
	require.NoError(t, db.SaveBlock(ctx, block), "Could not save genesis block")
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err, "Could not get signing root")
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
	require.NoError(t, state.SetValidators(validators))
	require.NoError(t, state.SetSlot(currentSlot))
	require.NoError(t, db.SaveState(ctx, state, genesisRoot), "Could not save state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, genesisRoot), "Could not save genesis state")

	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")
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
	require.NoError(t, err, "Could not get validator status")
	assert.Equal(t, ethpb.ValidatorStatus_PENDING, resp.Status)
	assert.Equal(t, uint64(2), resp.PositionInActivationQueue, "Unexpected position in activation queue")
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
	require.NoError(t, err, "Could not setup deposit trie")
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
	require.NoError(t, db.SaveBlock(ctx, block), "Could not save genesis block")
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err, "Could not get signing root")

	activeEpoch := helpers.ActivationExitEpoch(0)

	state, err := stateTrie.InitializeFromProtoUnsafe(&pbp2p.BeaconState{
		GenesisTime: uint64(time.Unix(0, 0).Unix()),
		Slot:        10000,
		Validators: []*ethpb.Validator{{
			ActivationEpoch: activeEpoch,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			PublicKey:       pubKey},
		}},
	)
	require.NoError(t, err)

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
	require.NoError(t, err, "Could not get the deposit block slot")
	assert.Equal(t, uint64(69), resp)
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
	require.NoError(t, err, "Could not setup deposit trie")
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
	require.NoError(t, db.SaveBlock(ctx, block), "Could not save genesis block")
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err, "Could not get signing root")

	activeEpoch := helpers.ActivationExitEpoch(0)

	state, err := stateTrie.InitializeFromProtoUnsafe(&pbp2p.BeaconState{
		GenesisTime: uint64(time.Unix(25000, 0).Unix()),
		Slot:        10000,
		Validators: []*ethpb.Validator{{
			ActivationEpoch: activeEpoch,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			PublicKey:       pubKey},
		}},
	)
	require.NoError(t, err)

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
	require.NoError(t, err, "Could not get the deposit block slot")
	assert.Equal(t, uint64(0), resp)
}

func TestMultipleValidatorStatus_Pubkeys(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	pubKeys := [][]byte{pubKey(1), pubKey(2), pubKey(3), pubKey(4)}
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
	require.NoError(t, err, "Could not get signing root")
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey(1),
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
		Amount:                10,
	}

	dep := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, dep, 10 /*blockNum*/, 0, depositTrie.Root())
	depData = &ethpb.Deposit_Data{
		PublicKey:             pubKey(3),
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
		Amount:                10,
	}

	dep = &ethpb.Deposit{
		Data: depData,
	}
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
	require.NoError(t, err)

	assert.Equal(t, len(response.PublicKeys), len(pubKeys))
	for i, resp := range response.PublicKeys {
		require.DeepEqual(t, pubKeys[i], resp)
	}
	assert.Equal(t, len(pubKeys), len(response.Statuses))
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
	require.NoError(t, err, "Could not get signing root")

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
	require.NoError(t, err)

	assert.Equal(t, len(beaconState.Validators), len(response.PublicKeys))
	for i, resp := range response.PublicKeys {
		expected := beaconState.Validators[i].PublicKey
		require.DeepEqual(t, expected, resp)
	}
	assert.Equal(t, len(beaconState.Validators), len(response.Statuses))
	for i, resp := range response.Statuses {
		if !proto.Equal(want[i], resp) {
			t.Fatalf("Wanted %v\n Recieved: %v\n", want[i], resp)
		}
	}
}

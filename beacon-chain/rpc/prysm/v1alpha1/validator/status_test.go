package validator

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/d4l3k/messagediff"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/time/slots"
	"google.golang.org/protobuf/proto"
)

func TestValidatorStatus_DepositedEth1(t *testing.T) {
	ctx := context.Background()
	deposits, _, err := util.DeterministicDepositsAndKeys(1)
	require.NoError(t, err, "Could not generate deposits and keys")
	deposit := deposits[0]
	pubKey1 := deposit.Data.PublicKey
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	assert.NoError(t, depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.HashTreeRoot()))
	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	stateObj, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{})
	require.NoError(t, err)
	vs := &Server{
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
	ctx := context.Background()

	pubKey1 := pubKey(1)
	depData := &ethpb.Deposit_Data{
		Amount:                params.BeaconConfig().MaxEffectiveBalance,
		PublicKey:             pubKey1,
		Signature:             bytesutil.PadTo([]byte("hi"), 96),
		WithdrawalCredentials: bytesutil.PadTo([]byte("hey"), 32),
	}
	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	assert.NoError(t, depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.HashTreeRoot()))
	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	stateObj, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{
		Validators: []*ethpb.Validator{
			{
				PublicKey:                  pubKey1,
				ActivationEligibilityEpoch: 1,
			},
		},
	})
	require.NoError(t, err)
	vs := &Server{
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

func TestValidatorStatus_PartiallyDeposited(t *testing.T) {
	ctx := context.Background()

	pubKey1 := pubKey(1)
	depData := &ethpb.Deposit_Data{
		Amount:                params.BeaconConfig().MinDepositAmount,
		PublicKey:             pubKey1,
		Signature:             []byte("hi"),
		WithdrawalCredentials: []byte("hey"),
	}
	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	assert.NoError(t, depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.HashTreeRoot()))
	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	stateObj, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{
		Validators: []*ethpb.Validator{
			{
				PublicKey:                  pubKey1,
				ActivationEligibilityEpoch: 1,
			},
		},
	})
	require.NoError(t, err)
	vs := &Server{
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
	assert.Equal(t, ethpb.ValidatorStatus_PARTIALLY_DEPOSITED, resp.Status)
}

func TestValidatorStatus_Pending(t *testing.T) {
	ctx := context.Background()

	pubKey := pubKey(1)
	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	// Pending active because activation epoch is still defaulted at far future slot.
	state, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, state.SetSlot(5000))
	err = state.SetValidators([]*ethpb.Validator{
		{
			ActivationEpoch:       params.BeaconConfig().FarFutureEpoch,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch:     params.BeaconConfig().FarFutureEpoch,
			PublicKey:             pubKey,
			WithdrawalCredentials: make([]byte, 32),
		},
	})
	require.NoError(t, err)

	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             bytesutil.PadTo([]byte("hi"), 96),
		WithdrawalCredentials: bytesutil.PadTo([]byte("hey"), 32),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	assert.NoError(t, depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.HashTreeRoot()))

	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	vs := &Server{
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
	// This test breaks if it doesnt use mainnet config
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	ctx := context.Background()

	pubKey := pubKey(1)

	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             bytesutil.PadTo([]byte("hi"), 96),
		WithdrawalCredentials: bytesutil.PadTo([]byte("hey"), 32),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	assert.NoError(t, depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.HashTreeRoot()))

	// Active because activation epoch <= current epoch < exit epoch.
	activeEpoch := helpers.ActivationExitEpoch(0)

	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")

	state := &ethpb.BeaconState{
		GenesisTime: uint64(time.Unix(0, 0).Unix()),
		Slot:        10000,
		Validators: []*ethpb.Validator{{
			ActivationEpoch:   activeEpoch,
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey:         pubKey},
		}}
	stateObj, err := v1.InitializeFromProtoUnsafe(state)
	require.NoError(t, err)

	timestamp := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			int(params.BeaconConfig().Eth1FollowDistance): uint64(timestamp),
		},
	}
	vs := &Server{
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
	ctx := context.Background()

	pubKey := pubKey(1)

	// Initiated exit because validator exit epoch and withdrawable epoch are not FAR_FUTURE_EPOCH
	slot := types.Slot(10000)
	epoch := slots.ToEpoch(slot)
	exitEpoch := helpers.ActivationExitEpoch(epoch)
	withdrawableEpoch := exitEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")

	state := &ethpb.BeaconState{
		Slot: slot,
		Validators: []*ethpb.Validator{{
			PublicKey:         pubKey,
			ActivationEpoch:   0,
			ExitEpoch:         exitEpoch,
			WithdrawableEpoch: withdrawableEpoch},
		}}
	stateObj, err := v1.InitializeFromProtoUnsafe(state)
	require.NoError(t, err)
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             bytesutil.PadTo([]byte("hi"), 96),
		WithdrawalCredentials: bytesutil.PadTo([]byte("hey"), 32),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	assert.NoError(t, depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.HashTreeRoot()))
	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	vs := &Server{
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
	ctx := context.Background()

	pubKey := pubKey(1)

	// Exit slashed because slashed is true, exit epoch is =< current epoch and withdrawable epoch > epoch .
	slot := types.Slot(10000)
	epoch := slots.ToEpoch(slot)
	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")

	state := &ethpb.BeaconState{
		Slot: slot,
		Validators: []*ethpb.Validator{{
			Slashed:           true,
			PublicKey:         pubKey,
			WithdrawableEpoch: epoch + 1},
		}}
	stateObj, err := v1.InitializeFromProtoUnsafe(state)
	require.NoError(t, err)
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             bytesutil.PadTo([]byte("hi"), 96),
		WithdrawalCredentials: bytesutil.PadTo([]byte("hey"), 32),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	assert.NoError(t, depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.HashTreeRoot()))
	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	vs := &Server{
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
	ctx := context.Background()

	pubKey := pubKey(1)

	// Exit because only exit epoch is =< current epoch.
	slot := types.Slot(10000)
	epoch := slots.ToEpoch(slot)
	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	state, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, state.SetSlot(slot))
	err = state.SetValidators([]*ethpb.Validator{{
		PublicKey:             pubKey,
		WithdrawableEpoch:     epoch + 1,
		WithdrawalCredentials: make([]byte, 32)},
	})
	require.NoError(t, err)
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		Signature:             bytesutil.PadTo([]byte("hi"), 96),
		WithdrawalCredentials: bytesutil.PadTo([]byte("hey"), 32),
	}

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	assert.NoError(t, depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.HashTreeRoot()))
	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	vs := &Server{
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
	pubKey := pubKey(1)
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	stateObj, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{
		Slot: 0,
	})
	require.NoError(t, err)
	vs := &Server{
		DepositFetcher:  depositCache,
		Eth1InfoFetcher: &mockPOW.POWChain{},
		HeadFetcher: &mockChain.ChainService{
			State: stateObj,
		},
	}
	req := &ethpb.ValidatorStatusRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	require.NoError(t, err, "Could not get validator status")
	assert.Equal(t, ethpb.ValidatorStatus_UNKNOWN_STATUS, resp.Status)
}

func TestActivationStatus_OK(t *testing.T) {
	ctx := context.Background()

	deposits, _, err := util.DeterministicDepositsAndKeys(4)
	require.NoError(t, err)
	pubKeys := [][]byte{deposits[0].Data.PublicKey, deposits[1].Data.PublicKey, deposits[2].Data.PublicKey, deposits[3].Data.PublicKey}
	stateObj, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{
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
				EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			},
		},
	})
	require.NoError(t, err)
	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	dep := deposits[0]
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	assert.NoError(t, depositCache.InsertDeposit(ctx, dep, 10 /*blockNum*/, 0, depositTrie.HashTreeRoot()))

	dep = deposits[2]
	assert.NoError(t, depositTrie.Insert(dep.Data.Signature, 15))
	assert.NoError(t, depositCache.InsertDeposit(context.Background(), dep, 0, 1, depositTrie.HashTreeRoot()))

	vs := &Server{
		Ctx:                context.Background(),
		CanonicalStateChan: make(chan *ethpb.BeaconState, 1),
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
	if uint64(response[2].Index) != uint64(params.BeaconConfig().FarFutureEpoch) {
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
	ctx := context.Background()

	pbKey := pubKey(5)
	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	currentSlot := types.Slot(5000)
	// Pending active because activation epoch is still defaulted at far future slot.
	validators := []*ethpb.Validator{
		{
			ActivationEpoch:       0,
			PublicKey:             pubKey(0),
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch:     params.BeaconConfig().FarFutureEpoch,
		},
		{
			ActivationEpoch:       0,
			PublicKey:             pubKey(1),
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch:     params.BeaconConfig().FarFutureEpoch,
		},
		{
			ActivationEpoch:       0,
			PublicKey:             pubKey(2),
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch:     params.BeaconConfig().FarFutureEpoch,
		},
		{
			ActivationEpoch:       0,
			PublicKey:             pubKey(3),
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch:     params.BeaconConfig().FarFutureEpoch,
		},
		{
			ActivationEpoch:       types.Epoch(currentSlot/params.BeaconConfig().SlotsPerEpoch + 1),
			PublicKey:             pbKey,
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch:     params.BeaconConfig().FarFutureEpoch,
		},
		{
			ActivationEpoch:       types.Epoch(currentSlot/params.BeaconConfig().SlotsPerEpoch + 4),
			PublicKey:             pubKey(5),
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch:     params.BeaconConfig().FarFutureEpoch,
		},
	}
	state, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, state.SetValidators(validators))
	require.NoError(t, state.SetSlot(currentSlot))

	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	for i := 0; i < 6; i++ {
		depData := &ethpb.Deposit_Data{
			PublicKey:             pubKey(uint64(i)),
			Signature:             bytesutil.PadTo([]byte("hi"), 96),
			WithdrawalCredentials: bytesutil.PadTo([]byte("hey"), 32),
		}

		deposit := &ethpb.Deposit{
			Data: depData,
		}
		assert.NoError(t, depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, int64(i), depositTrie.HashTreeRoot()))

	}

	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	vs := &Server{
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

func TestMultipleValidatorStatus_Pubkeys(t *testing.T) {
	ctx := context.Background()

	deposits, _, err := util.DeterministicDepositsAndKeys(6)
	require.NoError(t, err)
	pubKeys := [][]byte{
		deposits[0].Data.PublicKey,
		deposits[1].Data.PublicKey,
		deposits[2].Data.PublicKey,
		deposits[3].Data.PublicKey,
		deposits[4].Data.PublicKey,
		deposits[5].Data.PublicKey,
	}
	stateObj, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{
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
				EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			},
			{
				ActivationEligibilityEpoch: 700,
				ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
				PublicKey:                  pubKeys[4],
				EffectiveBalance:           params.BeaconConfig().MinDepositAmount,
			},
			{
				ActivationEligibilityEpoch: 700,
				ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
				PublicKey:                  pubKeys[5],
			},
		},
	})
	require.NoError(t, err)
	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	dep := deposits[0]
	assert.NoError(t, depositCache.InsertDeposit(ctx, dep, 10 /*blockNum*/, 0, depositTrie.HashTreeRoot()))
	dep = deposits[2]
	assert.NoError(t, depositTrie.Insert(dep.Data.Signature, 15))
	assert.NoError(t, depositCache.InsertDeposit(context.Background(), dep, 0, 1, depositTrie.HashTreeRoot()))

	vs := &Server{
		Ctx:                context.Background(),
		CanonicalStateChan: make(chan *ethpb.BeaconState, 1),
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
			Status:          ethpb.ValidatorStatus_DEPOSITED,
			ActivationEpoch: 18446744073709551615,
		},
		{
			Status: ethpb.ValidatorStatus_DEPOSITED,
		},
		{
			Status: ethpb.ValidatorStatus_PARTIALLY_DEPOSITED,
		},
		{
			Status: ethpb.ValidatorStatus_PENDING,
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
	slot := types.Slot(10000)
	epoch := slots.ToEpoch(slot)
	pubKeys := [][]byte{pubKey(1), pubKey(2), pubKey(3), pubKey(4), pubKey(5), pubKey(6), pubKey(7)}
	beaconState := &ethpb.BeaconState{
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
				EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			},
			{
				Slashed:   true,
				ExitEpoch: epoch + 1,
				PublicKey: pubKeys[3],
			},
			{
				ActivationEligibilityEpoch: 700,
				ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
				PublicKey:                  pubKeys[4],
				EffectiveBalance:           params.BeaconConfig().MinDepositAmount,
			},
			{
				ActivationEligibilityEpoch: 700,
				ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
				PublicKey:                  pubKeys[5],
			},
		},
	}
	stateObj, err := v1.InitializeFromProtoUnsafe(beaconState)
	require.NoError(t, err)
	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")

	vs := &Server{
		Ctx:                context.Background(),
		CanonicalStateChan: make(chan *ethpb.BeaconState, 1),
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
		{
			Status: ethpb.ValidatorStatus_PARTIALLY_DEPOSITED,
		},
		{
			Status: ethpb.ValidatorStatus_PENDING,
		},
	}

	// Note: Index 6 should be skipped.
	req := &ethpb.MultipleValidatorStatusRequest{Indices: []int64{0, 1, 2, 3, 4, 5, 6}}
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

func TestValidatorStatus_Invalid(t *testing.T) {
	ctx := context.Background()
	deposits, _, err := util.DeterministicDepositsAndKeys(1)
	require.NoError(t, err, "Could not generate deposits and keys")
	deposit := deposits[0]
	pubKey1 := deposit.Data.PublicKey
	deposit.Data.Signature = deposit.Data.Signature[1:]
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	assert.NoError(t, depositCache.InsertDeposit(ctx, deposit, 0 /*blockNum*/, 0, depositTrie.HashTreeRoot()))
	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}
	stateObj, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{})
	require.NoError(t, err)
	vs := &Server{
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
	assert.Equal(t, ethpb.ValidatorStatus_INVALID, resp.Status)
}

func Test_DepositStatus(t *testing.T) {
	assert.Equal(t, depositStatus(0), ethpb.ValidatorStatus_PENDING)
	assert.Equal(t, depositStatus(params.BeaconConfig().MinDepositAmount), ethpb.ValidatorStatus_PARTIALLY_DEPOSITED)
	assert.Equal(t, depositStatus(params.BeaconConfig().MaxEffectiveBalance), ethpb.ValidatorStatus_DEPOSITED)
}

func TestServer_CheckDoppelGanger(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		svSetup func(t *testing.T) (*Server, *ethpb.DoppelGangerRequest, *ethpb.DoppelGangerResponse)
	}{
		{
			name:    "normal doppelganger request",
			wantErr: false,
			svSetup: func(t *testing.T) (*Server, *ethpb.DoppelGangerRequest, *ethpb.DoppelGangerResponse) {
				mockGen := stategen.NewMockService()
				hs, ps, os, keys := createStateSetup(t, 4, mockGen)
				// Previous Epoch State
				for i := 0; i < 3; i++ {
					bal, err := ps.BalanceAtIndex(types.ValidatorIndex(i))
					assert.NoError(t, err)
					// Add 100 gwei, to mock an inactivity leak
					assert.NoError(t, ps.UpdateBalancesAtIndex(types.ValidatorIndex(i), bal+1000000000))
				}
				// Older Epoch State
				for i := 0; i < 3; i++ {
					bal, err := os.BalanceAtIndex(types.ValidatorIndex(i))
					assert.NoError(t, err)
					// Add 200 gwei, to mock an inactivity leak
					assert.NoError(t, os.UpdateBalancesAtIndex(types.ValidatorIndex(i), bal+2000000000))
				}
				vs := &Server{
					StateGen: mockGen,
					HeadFetcher: &mockChain.ChainService{
						State: hs,
					},
					SyncChecker: &mockSync.Sync{IsSyncing: false},
				}
				request := &ethpb.DoppelGangerRequest{
					ValidatorRequests: make([]*ethpb.DoppelGangerRequest_ValidatorRequest, 0),
				}
				response := &ethpb.DoppelGangerResponse{Responses: make([]*ethpb.DoppelGangerResponse_ValidatorResponse, 0)}
				for i := 0; i < 3; i++ {
					request.ValidatorRequests = append(request.ValidatorRequests, &ethpb.DoppelGangerRequest_ValidatorRequest{
						PublicKey:  keys[i].PublicKey().Marshal(),
						Epoch:      1,
						SignedRoot: []byte{'A'},
					})
					response.Responses = append(response.Responses, &ethpb.DoppelGangerResponse_ValidatorResponse{
						PublicKey:       keys[i].PublicKey().Marshal(),
						DuplicateExists: false,
					})
				}
				return vs, request, response
			},
		},
		{
			name:    "doppelganger exists current epoch",
			wantErr: false,
			svSetup: func(t *testing.T) (*Server, *ethpb.DoppelGangerRequest, *ethpb.DoppelGangerResponse) {
				mockGen := stategen.NewMockService()

				hs, ps, os, keys := createStateSetup(t, 4, mockGen)
				// Previous Epoch State
				for i := 0; i < 2; i++ {
					bal, err := ps.BalanceAtIndex(types.ValidatorIndex(i))
					assert.NoError(t, err)
					// Add 100 gwei, to mock an inactivity leak
					assert.NoError(t, ps.UpdateBalancesAtIndex(types.ValidatorIndex(i), bal+1000000000))
				}
				bal, err := ps.BalanceAtIndex(types.ValidatorIndex(2))
				assert.NoError(t, err)
				// Sub 100 gwei, to mock an active validator.
				assert.NoError(t, ps.UpdateBalancesAtIndex(types.ValidatorIndex(2), bal-1000000000))

				// Older Epoch State
				for i := 0; i < 2; i++ {
					bal, err := os.BalanceAtIndex(types.ValidatorIndex(i))
					assert.NoError(t, err)
					// Add 200 gwei, to mock an inactivity leak
					assert.NoError(t, os.UpdateBalancesAtIndex(types.ValidatorIndex(i), bal+2000000000))
				}
				bal, err = os.BalanceAtIndex(types.ValidatorIndex(2))
				assert.NoError(t, err)
				// Sub 100 gwei, to mock an active validator.
				assert.NoError(t, os.UpdateBalancesAtIndex(types.ValidatorIndex(2), bal-1000000000))

				vs := &Server{
					StateGen: mockGen,
					HeadFetcher: &mockChain.ChainService{
						State: hs,
					},
					SyncChecker: &mockSync.Sync{IsSyncing: false},
				}
				request := &ethpb.DoppelGangerRequest{
					ValidatorRequests: make([]*ethpb.DoppelGangerRequest_ValidatorRequest, 0),
				}
				response := &ethpb.DoppelGangerResponse{Responses: make([]*ethpb.DoppelGangerResponse_ValidatorResponse, 0)}
				for i := 0; i < 2; i++ {
					request.ValidatorRequests = append(request.ValidatorRequests, &ethpb.DoppelGangerRequest_ValidatorRequest{
						PublicKey:  keys[i].PublicKey().Marshal(),
						Epoch:      1,
						SignedRoot: []byte{'A'},
					})
					response.Responses = append(response.Responses, &ethpb.DoppelGangerResponse_ValidatorResponse{
						PublicKey:       keys[i].PublicKey().Marshal(),
						DuplicateExists: false,
					})
				}

				// Add in for duplicate validator
				request.ValidatorRequests = append(request.ValidatorRequests, &ethpb.DoppelGangerRequest_ValidatorRequest{
					PublicKey:  keys[2].PublicKey().Marshal(),
					Epoch:      1,
					SignedRoot: []byte{'A'},
				})
				response.Responses = append(response.Responses, &ethpb.DoppelGangerResponse_ValidatorResponse{
					PublicKey:       keys[2].PublicKey().Marshal(),
					DuplicateExists: true,
				})
				return vs, request, response
			},
		},
		{
			name:    "doppelganger exists previous epoch",
			wantErr: false,
			svSetup: func(t *testing.T) (*Server, *ethpb.DoppelGangerRequest, *ethpb.DoppelGangerResponse) {
				mockGen := stategen.NewMockService()

				hs, ps, os, keys := createStateSetup(t, 4, mockGen)
				// Previous Epoch State
				for i := 0; i < 2; i++ {
					bal, err := ps.BalanceAtIndex(types.ValidatorIndex(i))
					assert.NoError(t, err)
					// Add 100 gwei, to mock an inactivity leak
					assert.NoError(t, ps.UpdateBalancesAtIndex(types.ValidatorIndex(i), bal+1000000000))
				}
				bal, err := ps.BalanceAtIndex(types.ValidatorIndex(2))
				assert.NoError(t, err)
				// Sub 100 gwei, to mock an active validator.
				assert.NoError(t, ps.UpdateBalancesAtIndex(types.ValidatorIndex(2), bal-1000000000))

				// Older Epoch State
				for i := 0; i < 2; i++ {
					bal, err := os.BalanceAtIndex(types.ValidatorIndex(i))
					assert.NoError(t, err)
					// Add 200 gwei, to mock an inactivity leak
					assert.NoError(t, os.UpdateBalancesAtIndex(types.ValidatorIndex(i), bal+2000000000))
				}
				bal, err = os.BalanceAtIndex(types.ValidatorIndex(2))
				assert.NoError(t, err)
				// Sub 200 gwei, to mock an active validator.
				assert.NoError(t, os.UpdateBalancesAtIndex(types.ValidatorIndex(2), bal-2000000000))

				vs := &Server{
					StateGen: mockGen,
					HeadFetcher: &mockChain.ChainService{
						State: hs,
					},
					SyncChecker: &mockSync.Sync{IsSyncing: false},
				}
				request := &ethpb.DoppelGangerRequest{
					ValidatorRequests: make([]*ethpb.DoppelGangerRequest_ValidatorRequest, 0),
				}
				response := &ethpb.DoppelGangerResponse{Responses: make([]*ethpb.DoppelGangerResponse_ValidatorResponse, 0)}
				for i := 0; i < 2; i++ {
					request.ValidatorRequests = append(request.ValidatorRequests, &ethpb.DoppelGangerRequest_ValidatorRequest{
						PublicKey:  keys[i].PublicKey().Marshal(),
						Epoch:      1,
						SignedRoot: []byte{'A'},
					})
					response.Responses = append(response.Responses, &ethpb.DoppelGangerResponse_ValidatorResponse{
						PublicKey:       keys[i].PublicKey().Marshal(),
						DuplicateExists: false,
					})
				}

				// Add in for duplicate validator
				request.ValidatorRequests = append(request.ValidatorRequests, &ethpb.DoppelGangerRequest_ValidatorRequest{
					PublicKey:  keys[2].PublicKey().Marshal(),
					Epoch:      1,
					SignedRoot: []byte{'A'},
				})
				response.Responses = append(response.Responses, &ethpb.DoppelGangerResponse_ValidatorResponse{
					PublicKey:       keys[2].PublicKey().Marshal(),
					DuplicateExists: true,
				})
				return vs, request, response
			},
		},
		{
			name:    "multiple doppelganger exists",
			wantErr: false,
			svSetup: func(t *testing.T) (*Server, *ethpb.DoppelGangerRequest, *ethpb.DoppelGangerResponse) {
				mockGen := stategen.NewMockService()

				hs, ps, os, keys := createStateSetup(t, 4, mockGen)
				// Previous Epoch State
				for i := 10; i < 15; i++ {
					bal, err := ps.BalanceAtIndex(types.ValidatorIndex(i))
					assert.NoError(t, err)
					// Add 100 gwei, to mock an inactivity leak
					assert.NoError(t, ps.UpdateBalancesAtIndex(types.ValidatorIndex(i), bal-1000000000))
				}

				// Older Epoch State
				for i := 10; i < 15; i++ {
					bal, err := os.BalanceAtIndex(types.ValidatorIndex(i))
					assert.NoError(t, err)
					// Add 200 gwei, to mock an inactivity leak
					assert.NoError(t, os.UpdateBalancesAtIndex(types.ValidatorIndex(i), bal-2000000000))
				}

				vs := &Server{
					StateGen: mockGen,
					HeadFetcher: &mockChain.ChainService{
						State: hs,
					},
					SyncChecker: &mockSync.Sync{IsSyncing: false},
				}
				request := &ethpb.DoppelGangerRequest{
					ValidatorRequests: make([]*ethpb.DoppelGangerRequest_ValidatorRequest, 0),
				}
				response := &ethpb.DoppelGangerResponse{Responses: make([]*ethpb.DoppelGangerResponse_ValidatorResponse, 0)}
				for i := 10; i < 15; i++ {
					// Add in for duplicate validator
					request.ValidatorRequests = append(request.ValidatorRequests, &ethpb.DoppelGangerRequest_ValidatorRequest{
						PublicKey:  keys[i].PublicKey().Marshal(),
						Epoch:      1,
						SignedRoot: []byte{'A'},
					})
					response.Responses = append(response.Responses, &ethpb.DoppelGangerResponse_ValidatorResponse{
						PublicKey:       keys[i].PublicKey().Marshal(),
						DuplicateExists: true,
					})
				}

				return vs, request, response
			},
		},
		{
			name:    "attesters are too recent",
			wantErr: false,
			svSetup: func(t *testing.T) (*Server, *ethpb.DoppelGangerRequest, *ethpb.DoppelGangerResponse) {
				mockGen := stategen.NewMockService()

				hs, ps, os, keys := createStateSetup(t, 4, mockGen)
				// Previous Epoch State
				for i := 10; i < 15; i++ {
					bal, err := ps.BalanceAtIndex(types.ValidatorIndex(i))
					assert.NoError(t, err)
					// Add 100 gwei, to mock an active validator
					assert.NoError(t, ps.UpdateBalancesAtIndex(types.ValidatorIndex(i), bal-1000000000))
				}

				// Older Epoch State
				for i := 10; i < 15; i++ {
					bal, err := os.BalanceAtIndex(types.ValidatorIndex(i))
					assert.NoError(t, err)
					// Add 200 gwei, to mock an active validator
					assert.NoError(t, os.UpdateBalancesAtIndex(types.ValidatorIndex(i), bal-2000000000))
				}

				vs := &Server{
					StateGen: mockGen,
					HeadFetcher: &mockChain.ChainService{
						State: hs,
					},
					SyncChecker: &mockSync.Sync{IsSyncing: false},
				}
				request := &ethpb.DoppelGangerRequest{
					ValidatorRequests: make([]*ethpb.DoppelGangerRequest_ValidatorRequest, 0),
				}
				response := &ethpb.DoppelGangerResponse{Responses: make([]*ethpb.DoppelGangerResponse_ValidatorResponse, 0)}
				for i := 0; i < 15; i++ {
					request.ValidatorRequests = append(request.ValidatorRequests, &ethpb.DoppelGangerRequest_ValidatorRequest{
						PublicKey:  keys[i].PublicKey().Marshal(),
						Epoch:      2,
						SignedRoot: []byte{'A'},
					})
					response.Responses = append(response.Responses, &ethpb.DoppelGangerResponse_ValidatorResponse{
						PublicKey:       keys[i].PublicKey().Marshal(),
						DuplicateExists: false,
					})
				}

				return vs, request, response
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs, req, resp := tt.svSetup(t)
			got, err := vs.CheckDoppelGanger(context.Background(), req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckDoppelGanger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, resp) {
				diff, _ := messagediff.PrettyDiff(resp, got)
				t.Errorf("CheckDoppelGanger() difference = %v", diff)
			}
		})
	}
}

func createStateSetup(t *testing.T, head types.Epoch, mockgen *stategen.MockStateManager) (state.BeaconState,
	state.BeaconState, state.BeaconState, []bls.SecretKey) {
	gs, keys := util.DeterministicGenesisState(t, 64)
	hs := gs.Copy()
	// Head State
	headEpoch := head
	headSlot := types.Slot(headEpoch) * params.BeaconConfig().SlotsPerEpoch
	assert.NoError(t, hs.SetSlot(headSlot))
	assingments, _, err := helpers.CommitteeAssignments(context.Background(), hs, headEpoch)
	assert.NoError(t, err)
	for _, ctr := range assingments {
		pendingAtt := &ethpb.PendingAttestation{
			AggregationBits: bitfield.NewBitlist64(uint64(len(ctr.Committee))).ToBitlist().Not(),
			Data: &ethpb.AttestationData{
				Slot:            ctr.AttesterSlot,
				CommitteeIndex:  ctr.CommitteeIndex,
				BeaconBlockRoot: make([]byte, 32),
				Source: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  make([]byte, 32),
				},
				Target: &ethpb.Checkpoint{
					Epoch: 1,
					Root:  make([]byte, 32),
				},
			},
			InclusionDelay: 1,
			ProposerIndex:  10,
		}
		assert.NoError(t, hs.AppendCurrentEpochAttestations(pendingAtt))

	}
	mockgen.StatesBySlot[headSlot] = hs

	// Previous Epoch State
	prevEpoch := headEpoch - 1
	ps := gs.Copy()
	prevSlot, err := slots.EpochEnd(prevEpoch)
	assert.NoError(t, err)
	assert.NoError(t, ps.SetSlot(prevSlot))
	assingments, _, err = helpers.CommitteeAssignments(context.Background(), ps, prevEpoch)
	assert.NoError(t, err)
	for _, ctr := range assingments {
		pendingAtt := &ethpb.PendingAttestation{
			AggregationBits: bitfield.NewBitlist64(uint64(len(ctr.Committee))).ToBitlist().Not(),
			Data: &ethpb.AttestationData{
				Slot:            ctr.AttesterSlot,
				CommitteeIndex:  ctr.CommitteeIndex,
				BeaconBlockRoot: make([]byte, 32),
				Source: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  make([]byte, 32),
				},
				Target: &ethpb.Checkpoint{
					Epoch: 1,
					Root:  make([]byte, 32),
				},
			},
			InclusionDelay: 1,
			ProposerIndex:  10,
		}
		assert.NoError(t, ps.AppendCurrentEpochAttestations(pendingAtt))

	}
	mockgen.StatesBySlot[prevSlot] = ps

	// Older Epoch State
	olderEpoch := prevEpoch - 1
	os := gs.Copy()
	olderSlot, err := slots.EpochEnd(olderEpoch)
	assert.NoError(t, err)
	assert.NoError(t, os.SetSlot(olderSlot))
	assingments, _, err = helpers.CommitteeAssignments(context.Background(), os, olderEpoch)
	assert.NoError(t, err)
	for _, ctr := range assingments {
		attSlot := ctr.AttesterSlot
		if attSlot == olderSlot {
			continue
		}
		pendingAtt := &ethpb.PendingAttestation{
			AggregationBits: bitfield.NewBitlist64(uint64(len(ctr.Committee))).ToBitlist().Not(),
			Data: &ethpb.AttestationData{
				Slot:            attSlot,
				CommitteeIndex:  ctr.CommitteeIndex,
				BeaconBlockRoot: make([]byte, 32),
				Source: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  make([]byte, 32),
				},
				Target: &ethpb.Checkpoint{
					Epoch: 1,
					Root:  make([]byte, 32),
				},
			},
			InclusionDelay: 1,
			ProposerIndex:  10,
		}
		assert.NoError(t, os.AppendCurrentEpochAttestations(pendingAtt))

	}
	mockgen.StatesBySlot[olderSlot] = os
	return hs, ps, os, keys
}

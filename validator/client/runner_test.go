package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestCancelledContext_CleansUpValidator(t *testing.T) {
	v := &FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	run(cancelledContext(), v)
	assert.Equal(t, true, v.DoneCalled, "Expected Done() to be called")
}

func TestCancelledContext_WaitsForChainStart(t *testing.T) {
	v := &FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	run(cancelledContext(), v)
	assert.Equal(t, 1, v.WaitForChainStartCalled, "Expected WaitForChainStart() to be called")
}

func TestRetry_On_ConnectionError(t *testing.T) {
	retry := 10
	v := &FakeValidator{
		Keymanager:       &mockKeymanager{accountsChangedFeed: &event.Feed{}},
		RetryTillSuccess: retry,
	}
	backOffPeriod = 10 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	go run(ctx, v)
	// each step will fail (retry times)=10 this sleep times will wait more then
	// the time it takes for all steps to succeed before main loop.
	time.Sleep(time.Duration(retry*6) * backOffPeriod)
	cancel()
	// every call will fail retry=10 times so first one will be called 4 * retry=10.
	assert.Equal(t, retry*4, v.WaitForChainStartCalled, "Expected WaitForChainStart() to be called")
	assert.Equal(t, retry*3, v.WaitForSyncCalled, "Expected WaitForSync() to be called")
	assert.Equal(t, retry*2, v.WaitForActivationCalled, "Expected WaitForActivation() to be called")
	assert.Equal(t, retry, v.CanonicalHeadSlotCalled, "Expected WaitForActivation() to be called")
	assert.Equal(t, retry, v.ReceiveBlocksCalled, "Expected WaitForActivation() to be called")
}

func TestCancelledContext_WaitsForActivation(t *testing.T) {
	v := &FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	run(cancelledContext(), v)
	assert.Equal(t, 1, v.WaitForActivationCalled, "Expected WaitForActivation() to be called")
}

func TestCancelledContext_ChecksSlasherReady(t *testing.T) {
	v := &FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	cfg := &featureconfig.Flags{
		SlasherProtection: true,
	}
	reset := featureconfig.InitWithReset(cfg)
	defer reset()
	run(cancelledContext(), v)
	assert.Equal(t, true, v.SlasherReadyCalled, "Expected SlasherReady() to be called")
}

func TestUpdateDuties_NextSlot(t *testing.T) {
	v := &FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	ctx, cancel := context.WithCancel(context.Background())

	slot := types.Slot(55)
	ticker := make(chan types.Slot)
	v.NextSlotRet = ticker
	go func() {
		ticker <- slot

		cancel()
	}()

	run(ctx, v)

	require.Equal(t, true, v.UpdateDutiesCalled, "Expected UpdateAssignments(%d) to be called", slot)
	assert.Equal(t, uint64(slot), v.UpdateDutiesArg1, "UpdateAssignments was called with wrong argument")
}

func TestUpdateDuties_HandlesError(t *testing.T) {
	hook := logTest.NewGlobal()
	v := &FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	ctx, cancel := context.WithCancel(context.Background())

	slot := types.Slot(55)
	ticker := make(chan types.Slot)
	v.NextSlotRet = ticker
	go func() {
		ticker <- slot

		cancel()
	}()
	v.UpdateDutiesRet = errors.New("bad")

	run(ctx, v)

	require.LogsContain(t, hook, "Failed to update assignments")
}

func TestRoleAt_NextSlot(t *testing.T) {
	v := &FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	ctx, cancel := context.WithCancel(context.Background())

	slot := types.Slot(55)
	ticker := make(chan types.Slot)
	v.NextSlotRet = ticker
	go func() {
		ticker <- slot

		cancel()
	}()

	run(ctx, v)

	require.Equal(t, true, v.RoleAtCalled, "Expected RoleAt(%d) to be called", slot)
	assert.Equal(t, uint64(slot), v.RoleAtArg1, "RoleAt called with the wrong arg")
}

func TestAttests_NextSlot(t *testing.T) {
	v := &FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	ctx, cancel := context.WithCancel(context.Background())

	slot := types.Slot(55)
	ticker := make(chan types.Slot)
	v.NextSlotRet = ticker
	v.RolesAtRet = []ValidatorRole{roleAttester}
	go func() {
		ticker <- slot

		cancel()
	}()
	timer := time.NewTimer(200 * time.Millisecond)
	run(ctx, v)
	<-timer.C
	require.Equal(t, true, v.AttestToBlockHeadCalled, "SubmitAttestation(%d) was not called", slot)
	assert.Equal(t, uint64(slot), v.AttestToBlockHeadArg1, "SubmitAttestation was called with wrong arg")
}

func TestProposes_NextSlot(t *testing.T) {
	v := &FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	ctx, cancel := context.WithCancel(context.Background())

	slot := types.Slot(55)
	ticker := make(chan types.Slot)
	v.NextSlotRet = ticker
	v.RolesAtRet = []ValidatorRole{roleProposer}
	go func() {
		ticker <- slot

		cancel()
	}()
	timer := time.NewTimer(200 * time.Millisecond)
	run(ctx, v)
	<-timer.C
	require.Equal(t, true, v.ProposeBlockCalled, "ProposeBlock(%d) was not called", slot)
	assert.Equal(t, uint64(slot), v.ProposeBlockArg1, "ProposeBlock was called with wrong arg")
}

func TestBothProposesAndAttests_NextSlot(t *testing.T) {
	v := &FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	ctx, cancel := context.WithCancel(context.Background())

	slot := types.Slot(55)
	ticker := make(chan types.Slot)
	v.NextSlotRet = ticker
	v.RolesAtRet = []ValidatorRole{roleAttester, roleProposer}
	go func() {
		ticker <- slot

		cancel()
	}()
	timer := time.NewTimer(200 * time.Millisecond)
	run(ctx, v)
	<-timer.C
	require.Equal(t, true, v.AttestToBlockHeadCalled, "SubmitAttestation(%d) was not called", slot)
	assert.Equal(t, uint64(slot), v.AttestToBlockHeadArg1, "SubmitAttestation was called with wrong arg")
	require.Equal(t, true, v.ProposeBlockCalled, "ProposeBlock(%d) was not called", slot)
	assert.Equal(t, uint64(slot), v.ProposeBlockArg1, "ProposeBlock was called with wrong arg")
}

func TestAllValidatorsAreExited_NextSlot(t *testing.T) {
	v := &FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), allValidatorsAreExitedCtxKey, true))
	hook := logTest.NewGlobal()

	slot := types.Slot(55)
	ticker := make(chan types.Slot)
	v.NextSlotRet = ticker
	go func() {
		ticker <- slot

		cancel()
	}()
	run(ctx, v)
	assert.LogsContain(t, hook, "All validators are exited")
}

func TestHandleAccountsChanged_Ok(t *testing.T) {
	ctx := context.Background()
	defer ctx.Done()

	km := &mockKeymanager{accountsChangedFeed: &event.Feed{}}
	v := &FakeValidator{Keymanager: km}
	channel := make(chan struct{})
	go handleAccountsChanged(ctx, v, channel)
	time.Sleep(time.Second) // Allow time for subscribing to changes.
	km.SimulateAccountChanges()
	time.Sleep(time.Second) // Allow time for handling subscribed changes.

	select {
	case _, ok := <-channel:
		if !ok {
			t.Error("Account changed channel is closed")
		}
	default:
		t.Error("Accounts changed channel is empty")
	}
}

func TestHandleAccountsChanged_CtxCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	km := &mockKeymanager{accountsChangedFeed: &event.Feed{}}
	v := &FakeValidator{Keymanager: km}
	channel := make(chan struct{}, 2)
	go handleAccountsChanged(ctx, v, channel)
	time.Sleep(time.Second) // Allow time for subscribing to changes.
	km.SimulateAccountChanges()
	time.Sleep(time.Second) // Allow time for handling subscribed changes.

	cancel()
	time.Sleep(time.Second) // Allow time for handling cancellation.
	km.SimulateAccountChanges()
	time.Sleep(time.Second) // Allow time for handling subscribed changes.

	var values int
	for loop := true; loop == true; {
		select {
		case _, ok := <-channel:
			if ok {
				values++
			}
		default:
			loop = false
		}
	}
	assert.Equal(t, 1, values, "Incorrect number of values were passed to the channel")
}

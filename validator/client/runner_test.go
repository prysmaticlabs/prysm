package client

import (
	"context"
	"errors"
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/client/iface"
	"github.com/prysmaticlabs/prysm/validator/client/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestCancelledContext_CleansUpValidator(t *testing.T) {
	v := &testutil.FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	run(cancelledContext(), v)
	assert.Equal(t, true, v.DoneCalled, "Expected Done() to be called")
}

func TestCancelledContext_WaitsForChainStart(t *testing.T) {
	v := &testutil.FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	run(cancelledContext(), v)
	assert.Equal(t, 1, v.WaitForChainStartCalled, "Expected WaitForChainStart() to be called")
}

func TestRetry_On_ConnectionError(t *testing.T) {
	retry := 10
	v := &testutil.FakeValidator{
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
	v := &testutil.FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	run(cancelledContext(), v)
	assert.Equal(t, 1, v.WaitForActivationCalled, "Expected WaitForActivation() to be called")
}

func TestCancelledContext_ChecksSlasherReady(t *testing.T) {
	v := &testutil.FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	cfg := &featureconfig.Flags{
		SlasherProtection: true,
	}
	reset := featureconfig.InitWithReset(cfg)
	defer reset()
	run(cancelledContext(), v)
	assert.Equal(t, true, v.SlasherReadyCalled, "Expected SlasherReady() to be called")
}

func TestUpdateDuties_NextSlot(t *testing.T) {
	v := &testutil.FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
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
	v := &testutil.FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
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
	v := &testutil.FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
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
	v := &testutil.FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	ctx, cancel := context.WithCancel(context.Background())

	slot := types.Slot(55)
	ticker := make(chan types.Slot)
	v.NextSlotRet = ticker
	v.RolesAtRet = []iface.ValidatorRole{iface.RoleAttester}
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
	v := &testutil.FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	ctx, cancel := context.WithCancel(context.Background())

	slot := types.Slot(55)
	ticker := make(chan types.Slot)
	v.NextSlotRet = ticker
	v.RolesAtRet = []iface.ValidatorRole{iface.RoleProposer}
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
	v := &testutil.FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	ctx, cancel := context.WithCancel(context.Background())

	slot := types.Slot(55)
	ticker := make(chan types.Slot)
	v.NextSlotRet = ticker
	v.RolesAtRet = []iface.ValidatorRole{iface.RoleAttester, iface.RoleProposer}
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
	v := &testutil.FakeValidator{Keymanager: &mockKeymanager{accountsChangedFeed: &event.Feed{}}}
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), testutil.AllValidatorsAreExitedCtxKey, true))
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

func TestKeyReload_ActiveKey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	km := &mockKeymanager{}
	v := &testutil.FakeValidator{Keymanager: km}
	go func() {
		km.SimulateAccountChanges([][48]byte{testutil.ActiveKey})

		cancel()
	}()
	run(ctx, v)
	assert.Equal(t, true, v.HandleKeyReloadCalled)
	// We expect that WaitForActivation will only be called once,
	// at the very beginning, and not after account changes.
	assert.Equal(t, 1, v.WaitForActivationCalled)
}

func TestKeyReload_NoActiveKey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	km := &mockKeymanager{}
	v := &testutil.FakeValidator{Keymanager: km}
	go func() {
		km.SimulateAccountChanges(make([][48]byte, 0))

		cancel()
	}()
	run(ctx, v)
	assert.Equal(t, true, v.HandleKeyReloadCalled)
	assert.Equal(t, 2, v.WaitForActivationCalled)
}

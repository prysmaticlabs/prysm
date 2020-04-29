package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestCancelledContext_CleansUpValidator(t *testing.T) {
	v := &fakeValidator{}
	run(cancelledContext(), v)
	if !v.DoneCalled {
		t.Error("Expected Done() to be called")
	}
}

func TestCancelledContext_WaitsForChainStart(t *testing.T) {
	v := &fakeValidator{}
	run(cancelledContext(), v)
	if !v.WaitForChainStartCalled {
		t.Error("Expected WaitForChainStart() to be called")
	}
}

func TestCancelledContext_WaitsForSynced(t *testing.T) {
	cfg := &featureconfig.Flags{
		WaitForSynced: true,
	}
	reset := featureconfig.InitWithReset(cfg)
	defer reset()
	v := &fakeValidator{}
	run(cancelledContext(), v)
	if !v.WaitForSyncedCalled {
		t.Error("Expected WaitForSynced() to be called")
	}
}

func TestCancelledContext_WaitsForActivation(t *testing.T) {
	v := &fakeValidator{}
	run(cancelledContext(), v)
	if !v.WaitForActivationCalled {
		t.Error("Expected WaitForActivation() to be called")
	}
}

func TestUpdateDuties_NextSlot(t *testing.T) {
	v := &fakeValidator{}
	ctx, cancel := context.WithCancel(context.Background())

	slot := uint64(55)
	ticker := make(chan uint64)
	v.NextSlotRet = ticker
	go func() {
		ticker <- slot

		cancel()
	}()

	run(ctx, v)

	if !v.UpdateDutiesCalled {
		t.Fatalf("Expected UpdateAssignments(%d) to be called", slot)
	}
	if v.UpdateDutiesArg1 != slot {
		t.Errorf("UpdateAssignments was called with wrong argument. Want=%d, got=%d", slot, v.UpdateDutiesArg1)
	}
}

func TestUpdateDuties_HandlesError(t *testing.T) {
	hook := logTest.NewGlobal()
	v := &fakeValidator{}
	ctx, cancel := context.WithCancel(context.Background())

	slot := uint64(55)
	ticker := make(chan uint64)
	v.NextSlotRet = ticker
	go func() {
		ticker <- slot

		cancel()
	}()
	v.UpdateDutiesRet = errors.New("bad")

	run(ctx, v)

	testutil.AssertLogsContain(t, hook, "Failed to update assignments")
}

func TestRoleAt_NextSlot(t *testing.T) {
	v := &fakeValidator{}
	ctx, cancel := context.WithCancel(context.Background())

	slot := uint64(55)
	ticker := make(chan uint64)
	v.NextSlotRet = ticker
	go func() {
		ticker <- slot

		cancel()
	}()

	run(ctx, v)

	if !v.RoleAtCalled {
		t.Fatalf("Expected RoleAt(%d) to be called", slot)
	}
	if v.RoleAtArg1 != slot {
		t.Errorf("RoleAt called with the wrong arg. Want=%d, got=%d", slot, v.RoleAtArg1)
	}
}

func TestAttests_NextSlot(t *testing.T) {
	v := &fakeValidator{}
	ctx, cancel := context.WithCancel(context.Background())

	slot := uint64(55)
	ticker := make(chan uint64)
	v.NextSlotRet = ticker
	v.RolesAtRet = []validatorRole{roleAttester}
	go func() {
		ticker <- slot

		cancel()
	}()
	timer := time.NewTimer(200 * time.Millisecond)
	run(ctx, v)
	<-timer.C
	if !v.AttestToBlockHeadCalled {
		t.Fatalf("SubmitAttestation(%d) was not called", slot)
	}
	if v.AttestToBlockHeadArg1 != slot {
		t.Errorf("SubmitAttestation was called with wrong arg. Want=%d, got=%d", slot, v.AttestToBlockHeadArg1)
	}
}

func TestProposes_NextSlot(t *testing.T) {
	v := &fakeValidator{}
	ctx, cancel := context.WithCancel(context.Background())

	slot := uint64(55)
	ticker := make(chan uint64)
	v.NextSlotRet = ticker
	v.RolesAtRet = []validatorRole{roleProposer}
	go func() {
		ticker <- slot

		cancel()
	}()
	timer := time.NewTimer(200 * time.Millisecond)
	run(ctx, v)
	<-timer.C
	if !v.ProposeBlockCalled {
		t.Fatalf("ProposeBlock(%d) was not called", slot)
	}
	if v.ProposeBlockArg1 != slot {
		t.Errorf("ProposeBlock was called with wrong arg. Want=%d, got=%d", slot, v.AttestToBlockHeadArg1)
	}
}

func TestBothProposesAndAttests_NextSlot(t *testing.T) {
	v := &fakeValidator{}
	ctx, cancel := context.WithCancel(context.Background())

	slot := uint64(55)
	ticker := make(chan uint64)
	v.NextSlotRet = ticker
	v.RolesAtRet = []validatorRole{roleAttester, roleProposer}
	go func() {
		ticker <- slot

		cancel()
	}()
	timer := time.NewTimer(200 * time.Millisecond)
	run(ctx, v)
	<-timer.C
	if !v.AttestToBlockHeadCalled {
		t.Fatalf("SubmitAttestation(%d) was not called", slot)
	}
	if v.AttestToBlockHeadArg1 != slot {
		t.Errorf("SubmitAttestation was called with wrong arg. Want=%d, got=%d", slot, v.AttestToBlockHeadArg1)
	}
	if !v.ProposeBlockCalled {
		t.Fatalf("ProposeBlock(%d) was not called", slot)
	}
	if v.ProposeBlockArg1 != slot {
		t.Errorf("ProposeBlock was called with wrong arg. Want=%d, got=%d", slot, v.AttestToBlockHeadArg1)
	}
}

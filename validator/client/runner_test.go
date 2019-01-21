package client

import (
	"context"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
)

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestRun_initializesValidator(t *testing.T) {
	v := &fakeValidator{}
	run(cancelledContext(), v)
	if !v.InitializeCalled {
		t.Error("Expected Initialize() to be called")
	}
}

func TestRun_cleansUpValidator(t *testing.T) {
	v := &fakeValidator{}
	run(cancelledContext(), v)
	if !v.DoneCalled {
		t.Error("Expected Done() to be called")
	}
}

func TestRun_waitsForActivation(t *testing.T) {
	v := &fakeValidator{}
	run(cancelledContext(), v)
	if !v.WaitForActivationCalled {
		t.Error("Expected WaitForActivation() to be called")
	}
}

func TestRun_onNextSlot_updatesAssignments(t *testing.T) {
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

	if !v.UpdateAssignmentsCalled {
		t.Fatalf("Expected UpdateAssignments(%v) to be called", slot)
	}
	if v.UpdateAssignmentsArg1 != slot {
		t.Errorf("UpdateAssignments was called with wrong argument. Want=%v, got=%v", slot, v.UpdateAssignmentsArg1)
	}
}

func TestRun_onNextSlot_DeterminesRole(t *testing.T) {
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
		t.Fatalf("Expected RoleAt(%v) to be called", slot)
	}
	if v.RoleAtArg1 != slot {
		t.Errorf("RoleAt called with the wrong arg. Want=%v, got=%v", slot, v.RoleAtArg1)
	}
}

func TestRun_onNextSlot_actsAsAttester(t *testing.T) {
	v := &fakeValidator{}
	ctx, cancel := context.WithCancel(context.Background())

	slot := uint64(55)
	ticker := make(chan uint64)
	v.NextSlotRet = ticker
	v.RoleAtRet = pb.ValidatorRole_ATTESTER
	go func() {
		ticker <- slot

		cancel()
	}()

	run(ctx, v)

	if !v.AttestToBlockHeadCalled {
		t.Fatalf("AttestToBlockHead(%v) was not called", slot)
	}
	if v.AttestToBlockHeadArg1 != slot {
		t.Errorf("AttestToBlockHead was called with wrong arg. Want=%v, got=%v", slot, v.AttestToBlockHeadArg1)
	}
}

func TestRun_onNextSlot_actsAsProposer(t *testing.T) {
	v := &fakeValidator{}
	ctx, cancel := context.WithCancel(context.Background())

	slot := uint64(55)
	ticker := make(chan uint64)
	v.NextSlotRet = ticker
	v.RoleAtRet = pb.ValidatorRole_PROPOSER
	go func() {
		ticker <- slot

		cancel()
	}()

	run(ctx, v)

	if !v.ProposeBlockCalled {
		t.Fatalf("ProposeBlock(%v) was not called", slot)
	}
	if v.ProposeBlockArg1 != slot {
		t.Errorf("ProposeBlock was called with wrong arg. Want=%v, got=%v", slot, v.AttestToBlockHeadArg1)
	}
}

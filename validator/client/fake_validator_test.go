package client

import (
	"context"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
)

var _ = Validator(&fakeValidator{})

type fakeValidator struct {
	DoneCalled                       bool
	WaitForActivationCalled          bool
	WaitForChainStartCalled          bool
	NextSlotRet                      <-chan uint64
	NextSlotCalled                   bool
	UpdateAssignmentsCalled          bool
	UpdateAssignmentsArg1            uint64
	UpdateAssignmentsRet             error
	RoleAtCalled                     bool
	RoleAtArg1                       uint64
	RoleAtRet                        pb.ValidatorRole
	AttestToBlockHeadCalled          bool
	AttestToBlockHeadArg1            uint64
	ProposeBlockCalled               bool
	ProposeBlockArg1                 uint64
	LogValidatorGainsAndLossesCalled bool
}

func (fv *fakeValidator) Done() {
	fv.DoneCalled = true
}

func (fv *fakeValidator) WaitForChainStart(_ context.Context) error {
	fv.WaitForChainStartCalled = true
	return nil
}

func (fv *fakeValidator) WaitForActivation(_ context.Context) error {
	fv.WaitForActivationCalled = true
	return nil
}

func (fv *fakeValidator) NextSlot() <-chan uint64 {
	fv.NextSlotCalled = true
	return fv.NextSlotRet
}

func (fv *fakeValidator) UpdateAssignments(_ context.Context, slot uint64) error {
	fv.UpdateAssignmentsCalled = true
	fv.UpdateAssignmentsArg1 = slot
	return fv.UpdateAssignmentsRet
}

func (fv *fakeValidator) LogValidatorGainsAndLosses(_ context.Context, slot uint64) error {
	fv.LogValidatorGainsAndLossesCalled = true
	return nil
}

func (fv *fakeValidator) RoleAt(slot uint64) pb.ValidatorRole {
	fv.RoleAtCalled = true
	fv.RoleAtArg1 = slot
	return fv.RoleAtRet
}

func (fv *fakeValidator) AttestToBlockHead(_ context.Context, slot uint64) {
	fv.AttestToBlockHeadCalled = true
	fv.AttestToBlockHeadArg1 = slot
}

func (fv *fakeValidator) ProposeBlock(_ context.Context, slot uint64) {
	fv.ProposeBlockCalled = true
	fv.ProposeBlockArg1 = slot
}

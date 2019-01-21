package client

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
)

var _ = Validator(&fakeValidator{})

type fakeValidator struct {
	InitializeCalled bool

	DoneCalled bool

	WaitForActivationCalled bool

	NextSlotRet    <-chan uint64
	NextSlotCalled bool

	UpdateAssignmentsCalled bool
	UpdateAssignmentsArg1   uint64

	RoleAtCalled bool
	RoleAtArg1   uint64
	RoleAtRet    pb.ValidatorRole

	AttestToBlockHeadCalled bool
	AttestToBlockHeadArg1   uint64

	ProposeBlockCalled bool
	ProposeBlockArg1   uint64
}

func (fv *fakeValidator) Initialize() {
	fv.InitializeCalled = true
}

func (fv *fakeValidator) Done() {
	fv.DoneCalled = true
}

func (fv *fakeValidator) WaitForActivation() {
	fv.WaitForActivationCalled = true
}

func (fv *fakeValidator) NextSlot() <-chan uint64 {
	fv.NextSlotCalled = true
	return fv.NextSlotRet
}

func (fv *fakeValidator) UpdateAssignments(slot uint64) {
	fv.UpdateAssignmentsCalled = true
	fv.UpdateAssignmentsArg1 = slot
}

func (fv *fakeValidator) RoleAt(slot uint64) pb.ValidatorRole {
	fv.RoleAtCalled = true
	fv.RoleAtArg1 = slot
	return fv.RoleAtRet
}

func (fv *fakeValidator) AttestToBlockHead(slot uint64) {
	fv.AttestToBlockHeadCalled = true
	fv.AttestToBlockHeadArg1 = slot
}

func (fv *fakeValidator) ProposeBlock(slot uint64) {
	fv.ProposeBlockCalled = true
	fv.ProposeBlockArg1 = slot
}

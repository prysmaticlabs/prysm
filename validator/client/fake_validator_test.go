package client

import (
	"context"
	"time"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
)

var _ = Validator(&fakeValidator{})

type fakeValidator struct {
	DoneCalled                       bool
	WaitForActivationCalled          bool
	WaitForChainStartCalled          bool
	WaitForSyncCalled                bool
	NextSlotRet                      <-chan uint64
	NextSlotCalled                   bool
	CanonicalHeadSlotCalled          bool
	UpdateAssignmentsCalled          bool
	UpdateAssignmentsArg1            uint64
	UpdateAssignmentsRet             error
	RoleAtCalled                     bool
	RoleAtArg1                       uint64
	RolesAtRet                       []pb.ValidatorRole
	AttestToBlockHeadCalled          bool
	AttestToBlockHeadArg1            uint64
	ProposeBlockCalled               bool
	ProposeBlockArg1                 uint64
	LogValidatorGainsAndLossesCalled bool
	SlotDeadlineCalled               bool
	PublicKey                        string
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

func (fv *fakeValidator) WaitForSync(_ context.Context) error {
	fv.WaitForSyncCalled = true
	return nil
}

func (fv *fakeValidator) CanonicalHeadSlot(_ context.Context) (uint64, error) {
	fv.CanonicalHeadSlotCalled = true
	return 0, nil
}

func (fv *fakeValidator) SlotDeadline(_ uint64) time.Time {
	fv.SlotDeadlineCalled = true
	return time.Now()
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

func (fv *fakeValidator) RolesAt(slot uint64) map[[48]byte][]pb.ValidatorRole {
	fv.RoleAtCalled = true
	fv.RoleAtArg1 = slot
	vr := make(map[[48]byte][]pb.ValidatorRole)
	vr[[48]byte{1}] = fv.RolesAtRet
	return vr
}

func (fv *fakeValidator) SubmitAttestation(_ context.Context, slot uint64, pubKey [48]byte) {
	fv.AttestToBlockHeadCalled = true
	fv.AttestToBlockHeadArg1 = slot
}

func (fv *fakeValidator) ProposeBlock(_ context.Context, slot uint64, pubKey [48]byte) {
	fv.ProposeBlockCalled = true
	fv.ProposeBlockArg1 = slot
}

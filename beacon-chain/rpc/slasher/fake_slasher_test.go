package client

import (
	"context"
	"time"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
)

var _ = Validator(&fakeSlasher{})

type fakeSlasher struct {
	DoneCalled              bool
	WaitForActivationCalled bool
	WaitForSyncCalled       bool
}

func (fv *fakeSlasher) Done() {
	fv.DoneCalled = true
}

func (fv *fakeSlasher) WaitForChainStart(_ context.Context) error {
	fv.WaitForChainStartCalled = true
	return nil
}

func (fv *fakeSlasher) WaitForActivation(_ context.Context) error {
	fv.WaitForActivationCalled = true
	return nil
}

func (fv *fakeSlasher) WaitForSync(_ context.Context) error {
	fv.WaitForSyncCalled = true
	return nil
}

func (fv *fakeSlasher) CanonicalHeadSlot(_ context.Context) (uint64, error) {
	fv.CanonicalHeadSlotCalled = true
	return 0, nil
}

func (fv *fakeSlasher) SlotDeadline(_ uint64) time.Time {
	fv.SlotDeadlineCalled = true
	return time.Now()
}

func (fv *fakeSlasher) NextSlot() <-chan uint64 {
	fv.NextSlotCalled = true
	return fv.NextSlotRet
}

func (fv *fakeSlasher) UpdateDuties(_ context.Context, slot uint64) error {
	fv.UpdateDutiesCalled = true
	fv.UpdateDutiesArg1 = slot
	return fv.UpdateDutiesRet
}

func (fv *fakeSlasher) LogValidatorGainsAndLosses(_ context.Context, slot uint64) error {
	fv.LogValidatorGainsAndLossesCalled = true
	return nil
}

func (fv *fakeSlasher) RolesAt(_ context.Context, slot uint64) (map[[48]byte][]pb.ValidatorRole, error) {
	fv.RoleAtCalled = true
	fv.RoleAtArg1 = slot
	vr := make(map[[48]byte][]pb.ValidatorRole)
	vr[[48]byte{1}] = fv.RolesAtRet
	return vr, nil
}

func (fv *fakeSlasher) SubmitAttestation(_ context.Context, slot uint64, pubKey [48]byte) {
	fv.AttestToBlockHeadCalled = true
	fv.AttestToBlockHeadArg1 = slot
}

func (fv *fakeSlasher) ProposeBlock(_ context.Context, slot uint64, pubKey [48]byte) {
	fv.ProposeBlockCalled = true
	fv.ProposeBlockArg1 = slot
}

func (fv *fakeSlasher) SubmitAggregateAndProof(_ context.Context, slot uint64, pubKey [48]byte) {}

func (fv *fakeSlasher) LogAttestationsSubmitted() {}

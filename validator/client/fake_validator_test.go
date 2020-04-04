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
	NextSlotCalled                   bool
	CanonicalHeadSlotCalled          bool
	UpdateDutiesCalled               bool
	RoleAtCalled                     bool
	AttestToBlockHeadCalled          bool
	ProposeBlockCalled               bool
	LogValidatorGainsAndLossesCalled bool
	SlotDeadlineCalled               bool
	ProposeBlockArg1                 uint64
	AttestToBlockHeadArg1            uint64
	RoleAtArg1                       uint64
	UpdateDutiesArg1                 uint64
	NextSlotRet                      <-chan uint64
	PublicKey                        string
	UpdateDutiesRet                  error
	RolesAtRet                       []ValidatorRole
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

func (fv *fakeValidator) UpdateDuties(_ context.Context, slot uint64) error {
	fv.UpdateDutiesCalled = true
	fv.UpdateDutiesArg1 = slot
	return fv.UpdateDutiesRet
}

func (fv *fakeValidator) LogValidatorGainsAndLosses(_ context.Context, slot uint64) error {
	fv.LogValidatorGainsAndLossesCalled = true
	return nil
}

func (fv *fakeValidator) RolesAt(_ context.Context, slot uint64) (map[[48]byte][]ValidatorRole, error) {
	fv.RoleAtCalled = true
	fv.RoleAtArg1 = slot
	vr := make(map[[48]byte][]ValidatorRole)
	vr[[48]byte{1}] = fv.RolesAtRet
	return vr, nil
}

func (fv *fakeValidator) SubmitAttestation(_ context.Context, slot uint64, pubKey [48]byte) {
	fv.AttestToBlockHeadCalled = true
	fv.AttestToBlockHeadArg1 = slot
}

func (fv *fakeValidator) ProposeBlock(_ context.Context, slot uint64, pubKey [48]byte) {
	fv.ProposeBlockCalled = true
	fv.ProposeBlockArg1 = slot
}

func (fv *fakeValidator) SubmitAggregateAndProof(_ context.Context, slot uint64, pubKey [48]byte) {}

func (fv *fakeValidator) LogAttestationsSubmitted() {}

func (fv *fakeValidator) UpdateDomainDataCaches(context.Context, uint64) {}

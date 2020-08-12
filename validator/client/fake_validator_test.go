package client

import (
	"context"
	"time"
)

var _ = Validator(&fakeValidator{})

type fakeValidator struct {
	DoneCalled                       bool
	WaitForActivationCalled          bool
	WaitForChainStartCalled          bool
	WaitForSyncCalled                bool
	WaitForSyncedCalled              bool
	SlasherReadyCalled               bool
	NextSlotCalled                   bool
	CanonicalHeadSlotCalled          bool
	UpdateDutiesCalled               bool
	UpdateProtectionsCalled          bool
	RoleAtCalled                     bool
	AttestToBlockHeadCalled          bool
	ProposeBlockCalled               bool
	LogValidatorGainsAndLossesCalled bool
	SaveProtectionsCalled            bool
	SlotDeadlineCalled               bool
	ProposeBlockArg1                 uint64
	AttestToBlockHeadArg1            uint64
	RoleAtArg1                       uint64
	UpdateDutiesArg1                 uint64
	NextSlotRet                      <-chan uint64
	PublicKey                        string
	UpdateDutiesRet                  error
	RolesAtRet                       []validatorRole
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

func (fv *fakeValidator) WaitForSynced(_ context.Context) error {
	fv.WaitForSyncedCalled = true
	return nil
}

func (fv *fakeValidator) SlasherReady(_ context.Context) error {
	fv.SlasherReadyCalled = true
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

func (fv *fakeValidator) UpdateProtections(_ context.Context, slot uint64) error {
	fv.UpdateProtectionsCalled = true
	return nil
}

func (fv *fakeValidator) LogValidatorGainsAndLosses(_ context.Context, slot uint64) error {
	fv.LogValidatorGainsAndLossesCalled = true
	return nil
}

func (fv *fakeValidator) SaveProtections(_ context.Context) error {
	fv.SaveProtectionsCalled = true
	return nil
}

func (fv *fakeValidator) RolesAt(_ context.Context, slot uint64) (map[[48]byte][]validatorRole, error) {
	fv.RoleAtCalled = true
	fv.RoleAtArg1 = slot
	vr := make(map[[48]byte][]validatorRole)
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

func (fv *fakeValidator) BalancesByPubkeys(ctx context.Context) map[[48]byte]uint64 {
	return make(map[[48]byte]uint64)
}

func (fv *fakeValidator) PubKeysToIndices(ctx context.Context) map[uint64][48]byte {
	return make(map[uint64][48]byte)
}

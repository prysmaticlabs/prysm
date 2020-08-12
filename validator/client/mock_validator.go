package client

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

var _ = Validator(&FakeValidator{})

type FakeValidator struct {
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
	Balances                         map[[48]byte]uint64
	IndexToPubkeyMap                 map[uint64][48]byte
	PubkeyToIndexMap                 map[[48]byte]uint64
}

func (fv *FakeValidator) Done() {
	fv.DoneCalled = true
}

func (fv *FakeValidator) WaitForChainStart(_ context.Context) error {
	fv.WaitForChainStartCalled = true
	return nil
}

func (fv *FakeValidator) WaitForActivation(_ context.Context) error {
	fv.WaitForActivationCalled = true
	return nil
}

func (fv *FakeValidator) WaitForSync(_ context.Context) error {
	fv.WaitForSyncCalled = true
	return nil
}

func (fv *FakeValidator) WaitForSynced(_ context.Context) error {
	fv.WaitForSyncedCalled = true
	return nil
}

func (fv *FakeValidator) SlasherReady(_ context.Context) error {
	fv.SlasherReadyCalled = true
	return nil
}

func (fv *FakeValidator) CanonicalHeadSlot(_ context.Context) (uint64, error) {
	fv.CanonicalHeadSlotCalled = true
	return 0, nil
}

func (fv *FakeValidator) SlotDeadline(_ uint64) time.Time {
	fv.SlotDeadlineCalled = true
	return roughtime.Now()
}

func (fv *FakeValidator) NextSlot() <-chan uint64 {
	fv.NextSlotCalled = true
	return fv.NextSlotRet
}

func (fv *FakeValidator) UpdateDuties(_ context.Context, slot uint64) error {
	fv.UpdateDutiesCalled = true
	fv.UpdateDutiesArg1 = slot
	return fv.UpdateDutiesRet
}

func (fv *FakeValidator) UpdateProtections(_ context.Context, slot uint64) error {
	fv.UpdateProtectionsCalled = true
	return nil
}

func (fv *FakeValidator) LogValidatorGainsAndLosses(_ context.Context, slot uint64) error {
	fv.LogValidatorGainsAndLossesCalled = true
	return nil
}

func (fv *FakeValidator) SaveProtections(_ context.Context) error {
	fv.SaveProtectionsCalled = true
	return nil
}

func (fv *FakeValidator) RolesAt(_ context.Context, slot uint64) (map[[48]byte][]validatorRole, error) {
	fv.RoleAtCalled = true
	fv.RoleAtArg1 = slot
	vr := make(map[[48]byte][]validatorRole)
	vr[[48]byte{1}] = fv.RolesAtRet
	return vr, nil
}

func (fv *FakeValidator) SubmitAttestation(_ context.Context, slot uint64, pubKey [48]byte) {
	fv.AttestToBlockHeadCalled = true
	fv.AttestToBlockHeadArg1 = slot
}

func (fv *FakeValidator) ProposeBlock(_ context.Context, slot uint64, pubKey [48]byte) {
	fv.ProposeBlockCalled = true
	fv.ProposeBlockArg1 = slot
}

func (fv *FakeValidator) SubmitAggregateAndProof(_ context.Context, slot uint64, pubKey [48]byte) {}

func (fv *FakeValidator) LogAttestationsSubmitted() {}

func (fv *FakeValidator) UpdateDomainDataCaches(context.Context, uint64) {}

func (fv *FakeValidator) BalancesByPubkeys(ctx context.Context) map[[48]byte]uint64 {
	return fv.Balances
}

func (fv *FakeValidator) IndicesToPubkeys(ctx context.Context) map[uint64][48]byte {
	return fv.IndexToPubkeyMap
}

func (fv *FakeValidator) PubkeysToIndices(ctx context.Context) map[[48]byte]uint64 {
	return fv.PubkeyToIndexMap
}

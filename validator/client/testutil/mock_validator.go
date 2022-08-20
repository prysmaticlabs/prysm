package testutil

import (
	"bytes"
	"context"
	"time"

	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	prysmTime "github.com/prysmaticlabs/prysm/v3/time"
	"github.com/prysmaticlabs/prysm/v3/validator/client/iface"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	log "github.com/sirupsen/logrus"
)

var _ iface.Validator = (*FakeValidator)(nil)

// FakeValidator for mocking.
type FakeValidator struct {
	DoneCalled                        bool
	WaitForWalletInitializationCalled bool
	SlasherReadyCalled                bool
	NextSlotCalled                    bool
	UpdateDutiesCalled                bool
	UpdateProtectionsCalled           bool
	RoleAtCalled                      bool
	AttestToBlockHeadCalled           bool
	ProposeBlockCalled                bool
	LogValidatorGainsAndLossesCalled  bool
	SaveProtectionsCalled             bool
	DeleteProtectionCalled            bool
	SlotDeadlineCalled                bool
	HandleKeyReloadCalled             bool
	WaitForChainStartCalled           int
	WaitForSyncCalled                 int
	WaitForActivationCalled           int
	CanonicalHeadSlotCalled           int
	ReceiveBlocksCalled               int
	RetryTillSuccess                  int
	ProposeBlockArg1                  uint64
	AttestToBlockHeadArg1             uint64
	RoleAtArg1                        uint64
	UpdateDutiesArg1                  uint64
	NextSlotRet                       <-chan types.Slot
	PublicKey                         string
	UpdateDutiesRet                   error
	ProposerSettingsErr               error
	RolesAtRet                        []iface.ValidatorRole
	Balances                          map[[fieldparams.BLSPubkeyLength]byte]uint64
	IndexToPubkeyMap                  map[uint64][fieldparams.BLSPubkeyLength]byte
	PubkeyToIndexMap                  map[[fieldparams.BLSPubkeyLength]byte]uint64
	PubkeysToStatusesMap              map[[fieldparams.BLSPubkeyLength]byte]ethpb.ValidatorStatus
	Km                                keymanager.IKeymanager
}

type ctxKey string

// AllValidatorsAreExitedCtxKey represents the metadata context key used for exits.
var AllValidatorsAreExitedCtxKey = ctxKey("exited")

// Done for mocking.
func (fv *FakeValidator) Done() {
	fv.DoneCalled = true
}

// WaitForKeymanagerInitialization for mocking.
func (fv *FakeValidator) WaitForKeymanagerInitialization(_ context.Context) error {
	fv.WaitForWalletInitializationCalled = true
	return nil
}

// LogSyncCommitteeMessagesSubmitted --
func (fv *FakeValidator) LogSyncCommitteeMessagesSubmitted() {}

// WaitForChainStart for mocking.
func (fv *FakeValidator) WaitForChainStart(_ context.Context) error {
	fv.WaitForChainStartCalled++
	if fv.RetryTillSuccess >= fv.WaitForChainStartCalled {
		return iface.ErrConnectionIssue
	}
	return nil
}

// WaitForActivation for mocking.
func (fv *FakeValidator) WaitForActivation(_ context.Context, _ chan [][fieldparams.BLSPubkeyLength]byte) error {
	fv.WaitForActivationCalled++
	if fv.RetryTillSuccess >= fv.WaitForActivationCalled {
		return iface.ErrConnectionIssue
	}
	return nil
}

// WaitForSync for mocking.
func (fv *FakeValidator) WaitForSync(_ context.Context) error {
	fv.WaitForSyncCalled++
	if fv.RetryTillSuccess >= fv.WaitForSyncCalled {
		return iface.ErrConnectionIssue
	}
	return nil
}

// SlasherReady for mocking.
func (fv *FakeValidator) SlasherReady(_ context.Context) error {
	fv.SlasherReadyCalled = true
	return nil
}

// CanonicalHeadSlot for mocking.
func (fv *FakeValidator) CanonicalHeadSlot(_ context.Context) (types.Slot, error) {
	fv.CanonicalHeadSlotCalled++
	if fv.RetryTillSuccess > fv.CanonicalHeadSlotCalled {
		return 0, iface.ErrConnectionIssue
	}
	return 0, nil
}

// SlotDeadline for mocking.
func (fv *FakeValidator) SlotDeadline(_ types.Slot) time.Time {
	fv.SlotDeadlineCalled = true
	return prysmTime.Now()
}

// NextSlot for mocking.
func (fv *FakeValidator) NextSlot() <-chan types.Slot {
	fv.NextSlotCalled = true
	return fv.NextSlotRet
}

// UpdateDuties for mocking.
func (fv *FakeValidator) UpdateDuties(_ context.Context, slot types.Slot) error {
	fv.UpdateDutiesCalled = true
	fv.UpdateDutiesArg1 = uint64(slot)
	return fv.UpdateDutiesRet
}

// UpdateProtections for mocking.
func (fv *FakeValidator) UpdateProtections(_ context.Context, _ uint64) error {
	fv.UpdateProtectionsCalled = true
	return nil
}

// LogValidatorGainsAndLosses for mocking.
func (fv *FakeValidator) LogValidatorGainsAndLosses(_ context.Context, _ types.Slot) error {
	fv.LogValidatorGainsAndLossesCalled = true
	return nil
}

// ResetAttesterProtectionData for mocking.
func (fv *FakeValidator) ResetAttesterProtectionData() {
	fv.DeleteProtectionCalled = true
}

// RolesAt for mocking.
func (fv *FakeValidator) RolesAt(_ context.Context, slot types.Slot) (map[[fieldparams.BLSPubkeyLength]byte][]iface.ValidatorRole, error) {
	fv.RoleAtCalled = true
	fv.RoleAtArg1 = uint64(slot)
	vr := make(map[[fieldparams.BLSPubkeyLength]byte][]iface.ValidatorRole)
	vr[[fieldparams.BLSPubkeyLength]byte{1}] = fv.RolesAtRet
	return vr, nil
}

// SubmitAttestation for mocking.
func (fv *FakeValidator) SubmitAttestation(_ context.Context, slot types.Slot, _ [fieldparams.BLSPubkeyLength]byte) {
	fv.AttestToBlockHeadCalled = true
	fv.AttestToBlockHeadArg1 = uint64(slot)
}

// ProposeBlock for mocking.
func (fv *FakeValidator) ProposeBlock(_ context.Context, slot types.Slot, _ [fieldparams.BLSPubkeyLength]byte) {
	fv.ProposeBlockCalled = true
	fv.ProposeBlockArg1 = uint64(slot)
}

// SubmitAggregateAndProof for mocking.
func (_ *FakeValidator) SubmitAggregateAndProof(_ context.Context, _ types.Slot, _ [fieldparams.BLSPubkeyLength]byte) {
}

// SubmitSyncCommitteeMessage for mocking.
func (_ *FakeValidator) SubmitSyncCommitteeMessage(_ context.Context, _ types.Slot, _ [fieldparams.BLSPubkeyLength]byte) {
}

// LogAttestationsSubmitted for mocking.
func (_ *FakeValidator) LogAttestationsSubmitted() {}

// UpdateDomainDataCaches for mocking.
func (_ *FakeValidator) UpdateDomainDataCaches(context.Context, types.Slot) {}

// BalancesByPubkeys for mocking.
func (fv *FakeValidator) BalancesByPubkeys(_ context.Context) map[[fieldparams.BLSPubkeyLength]byte]uint64 {
	return fv.Balances
}

// IndicesToPubkeys for mocking.
func (fv *FakeValidator) IndicesToPubkeys(_ context.Context) map[uint64][fieldparams.BLSPubkeyLength]byte {
	return fv.IndexToPubkeyMap
}

// PubkeysToIndices for mocking.
func (fv *FakeValidator) PubkeysToIndices(_ context.Context) map[[fieldparams.BLSPubkeyLength]byte]uint64 {
	return fv.PubkeyToIndexMap
}

// PubkeysToStatuses for mocking.
func (fv *FakeValidator) PubkeysToStatuses(_ context.Context) map[[fieldparams.BLSPubkeyLength]byte]ethpb.ValidatorStatus {
	return fv.PubkeysToStatusesMap
}

// AllValidatorsAreExited for mocking
func (_ *FakeValidator) AllValidatorsAreExited(ctx context.Context) (bool, error) {
	if ctx.Value(AllValidatorsAreExitedCtxKey) == nil {
		return false, nil
	}
	return ctx.Value(AllValidatorsAreExitedCtxKey).(bool), nil
}

// Keymanager for mocking
func (fv *FakeValidator) Keymanager() (keymanager.IKeymanager, error) {
	return fv.Km, nil
}

// CheckDoppelGanger for mocking
func (_ *FakeValidator) CheckDoppelGanger(_ context.Context) error {
	return nil
}

// ReceiveBlocks for mocking
func (fv *FakeValidator) ReceiveBlocks(_ context.Context, connectionErrorChannel chan<- error) {
	fv.ReceiveBlocksCalled++
	if fv.RetryTillSuccess > fv.ReceiveBlocksCalled {
		connectionErrorChannel <- iface.ErrConnectionIssue
	}
}

// HandleKeyReload for mocking
func (fv *FakeValidator) HandleKeyReload(_ context.Context, newKeys [][fieldparams.BLSPubkeyLength]byte) (anyActive bool, err error) {
	fv.HandleKeyReloadCalled = true
	for _, key := range newKeys {
		if bytes.Equal(key[:], ActiveKey[:]) {
			return true, nil
		}
	}
	return false, nil
}

// SubmitSignedContributionAndProof for mocking
func (_ *FakeValidator) SubmitSignedContributionAndProof(_ context.Context, _ types.Slot, _ [fieldparams.BLSPubkeyLength]byte) {
}

// PushProposerSettings for mocking
func (fv *FakeValidator) PushProposerSettings(_ context.Context, _ keymanager.IKeymanager) error {
	if fv.ProposerSettingsErr != nil {
		return fv.ProposerSettingsErr
	}
	log.Infoln("Mock updated proposer settings")
	return nil
}

// SetPubKeyToValidatorIndexMap for mocking
func (_ *FakeValidator) SetPubKeyToValidatorIndexMap(_ context.Context, _ keymanager.IKeymanager) error {
	return nil
}

// SignValidatorRegistrationRequest for mocking
func (_ *FakeValidator) SignValidatorRegistrationRequest(_ context.Context, _ iface.SigningFunc, _ *ethpb.ValidatorRegistrationV1) (*ethpb.SignedValidatorRegistrationV1, error) {
	return nil, nil
}

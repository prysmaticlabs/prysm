package testutil

import (
	"bytes"
	"context"
	"time"

	api "github.com/prysmaticlabs/prysm/v5/api/client"
	"github.com/prysmaticlabs/prysm/v5/api/client/beacon"
	"github.com/prysmaticlabs/prysm/v5/api/client/event"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/proposer"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	prysmTime "github.com/prysmaticlabs/prysm/v5/time"
	"github.com/prysmaticlabs/prysm/v5/validator/client/iface"
	"github.com/prysmaticlabs/prysm/v5/validator/keymanager"
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
	NextSlotRet                       <-chan primitives.Slot
	PublicKey                         string
	UpdateDutiesRet                   error
	ProposerSettingsErr               error
	RolesAtRet                        []iface.ValidatorRole
	Balances                          map[[fieldparams.BLSPubkeyLength]byte]uint64
	IndexToPubkeyMap                  map[uint64][fieldparams.BLSPubkeyLength]byte
	PubkeyToIndexMap                  map[[fieldparams.BLSPubkeyLength]byte]uint64
	PubkeysToStatusesMap              map[[fieldparams.BLSPubkeyLength]byte]ethpb.ValidatorStatus
	proposerSettings                  *proposer.Settings
	ProposerSettingWait               time.Duration
	Km                                keymanager.IKeymanager
	graffiti                          string
	Tracker                           *beacon.NodeHealthTracker
}

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
func (fv *FakeValidator) LogSubmittedSyncCommitteeMessages() {}

// WaitForChainStart for mocking.
func (fv *FakeValidator) WaitForChainStart(_ context.Context) error {
	fv.WaitForChainStartCalled++
	if fv.RetryTillSuccess >= fv.WaitForChainStartCalled {
		return api.ErrConnectionIssue
	}
	return nil
}

// WaitForActivation for mocking.
func (fv *FakeValidator) WaitForActivation(_ context.Context, accountChan chan [][fieldparams.BLSPubkeyLength]byte) error {
	fv.WaitForActivationCalled++
	if accountChan == nil {
		return nil
	}
	if fv.RetryTillSuccess >= fv.WaitForActivationCalled {
		return api.ErrConnectionIssue
	}
	return nil
}

// WaitForSync for mocking.
func (fv *FakeValidator) WaitForSync(_ context.Context) error {
	fv.WaitForSyncCalled++
	if fv.RetryTillSuccess >= fv.WaitForSyncCalled {
		return api.ErrConnectionIssue
	}
	return nil
}

// SlasherReady for mocking.
func (fv *FakeValidator) SlasherReady(_ context.Context) error {
	fv.SlasherReadyCalled = true
	return nil
}

// CanonicalHeadSlot for mocking.
func (fv *FakeValidator) CanonicalHeadSlot(_ context.Context) (primitives.Slot, error) {
	fv.CanonicalHeadSlotCalled++
	if fv.RetryTillSuccess > fv.CanonicalHeadSlotCalled {
		return 0, api.ErrConnectionIssue
	}
	return 0, nil
}

// SlotDeadline for mocking.
func (fv *FakeValidator) SlotDeadline(_ primitives.Slot) time.Time {
	fv.SlotDeadlineCalled = true
	return prysmTime.Now()
}

// NextSlot for mocking.
func (fv *FakeValidator) NextSlot() <-chan primitives.Slot {
	fv.NextSlotCalled = true
	return fv.NextSlotRet
}

// UpdateDuties for mocking.
func (fv *FakeValidator) UpdateDuties(_ context.Context, slot primitives.Slot) error {
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
func (fv *FakeValidator) LogValidatorGainsAndLosses(_ context.Context, _ primitives.Slot) error {
	fv.LogValidatorGainsAndLossesCalled = true
	return nil
}

// ResetAttesterProtectionData for mocking.
func (fv *FakeValidator) ResetAttesterProtectionData() {
	fv.DeleteProtectionCalled = true
}

// RolesAt for mocking.
func (fv *FakeValidator) RolesAt(_ context.Context, slot primitives.Slot) (map[[fieldparams.BLSPubkeyLength]byte][]iface.ValidatorRole, error) {
	fv.RoleAtCalled = true
	fv.RoleAtArg1 = uint64(slot)
	vr := make(map[[fieldparams.BLSPubkeyLength]byte][]iface.ValidatorRole)
	vr[[fieldparams.BLSPubkeyLength]byte{1}] = fv.RolesAtRet
	return vr, nil
}

// SubmitAttestation for mocking.
func (fv *FakeValidator) SubmitAttestation(_ context.Context, slot primitives.Slot, _ [fieldparams.BLSPubkeyLength]byte) {
	fv.AttestToBlockHeadCalled = true
	fv.AttestToBlockHeadArg1 = uint64(slot)
}

// ProposeBlock for mocking.
func (fv *FakeValidator) ProposeBlock(_ context.Context, slot primitives.Slot, _ [fieldparams.BLSPubkeyLength]byte) {
	fv.ProposeBlockCalled = true
	fv.ProposeBlockArg1 = uint64(slot)
}

// SubmitAggregateAndProof for mocking.
func (*FakeValidator) SubmitAggregateAndProof(_ context.Context, _ primitives.Slot, _ [fieldparams.BLSPubkeyLength]byte) {
}

// SubmitSyncCommitteeMessage for mocking.
func (*FakeValidator) SubmitSyncCommitteeMessage(_ context.Context, _ primitives.Slot, _ [fieldparams.BLSPubkeyLength]byte) {
}

// LogSubmittedAtts for mocking.
func (*FakeValidator) LogSubmittedAtts(_ primitives.Slot) {}

// UpdateDomainDataCaches for mocking.
func (*FakeValidator) UpdateDomainDataCaches(context.Context, primitives.Slot) {}

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

// Keymanager for mocking
func (fv *FakeValidator) Keymanager() (keymanager.IKeymanager, error) {
	return fv.Km, nil
}

// CheckDoppelGanger for mocking
func (*FakeValidator) CheckDoppelGanger(_ context.Context) error {
	return nil
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
func (*FakeValidator) SubmitSignedContributionAndProof(_ context.Context, _ primitives.Slot, _ [fieldparams.BLSPubkeyLength]byte) {
}

// HasProposerSettings for mocking
func (*FakeValidator) HasProposerSettings() bool {
	return true
}

// PushProposerSettings for mocking
func (fv *FakeValidator) PushProposerSettings(ctx context.Context, km keymanager.IKeymanager, slot primitives.Slot, deadline time.Time) error {
	nctx, cancel := context.WithDeadline(ctx, deadline)
	ctx = nctx
	defer cancel()
	time.Sleep(fv.ProposerSettingWait)
	if ctx.Err() == context.DeadlineExceeded {
		log.Error("deadline exceeded")
		// can't return error or it will trigger a log.fatal
		return nil
	}

	if fv.ProposerSettingsErr != nil {
		return fv.ProposerSettingsErr
	}

	log.Infoln("Mock updated proposer settings")
	return nil
}

// SetPubKeyToValidatorIndexMap for mocking
func (*FakeValidator) SetPubKeyToValidatorIndexMap(_ context.Context, _ keymanager.IKeymanager) error {
	return nil
}

// SignValidatorRegistrationRequest for mocking
func (*FakeValidator) SignValidatorRegistrationRequest(_ context.Context, _ iface.SigningFunc, _ *ethpb.ValidatorRegistrationV1) (*ethpb.SignedValidatorRegistrationV1, error) {
	return nil, nil
}

// ProposerSettings for mocking
func (fv *FakeValidator) ProposerSettings() *proposer.Settings {
	return fv.proposerSettings
}

// SetProposerSettings for mocking
func (fv *FakeValidator) SetProposerSettings(_ context.Context, settings *proposer.Settings) error {
	fv.proposerSettings = settings
	return nil
}

// GetGraffiti for mocking
func (f *FakeValidator) GetGraffiti(_ context.Context, _ [fieldparams.BLSPubkeyLength]byte) ([]byte, error) {
	return []byte(f.graffiti), nil
}

// SetGraffiti for mocking
func (f *FakeValidator) SetGraffiti(_ context.Context, _ [fieldparams.BLSPubkeyLength]byte, graffiti []byte) error {
	f.graffiti = string(graffiti)
	return nil
}

// DeleteGraffiti for mocking
func (f *FakeValidator) DeleteGraffiti(_ context.Context, _ [fieldparams.BLSPubkeyLength]byte) error {
	f.graffiti = ""
	return nil
}

func (*FakeValidator) StartEventStream(_ context.Context, _ []string, _ chan<- *event.Event) {

}

func (*FakeValidator) ProcessEvent(_ *event.Event) {}

func (*FakeValidator) EventStreamIsRunning() bool {
	return true
}

func (fv *FakeValidator) HealthTracker() *beacon.NodeHealthTracker {
	return fv.Tracker
}

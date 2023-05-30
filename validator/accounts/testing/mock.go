package mock

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	validatorserviceconfig "github.com/prysmaticlabs/prysm/v4/config/validator/service"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/validator/accounts/iface"
	iface2 "github.com/prysmaticlabs/prysm/v4/validator/client/iface"
	"github.com/prysmaticlabs/prysm/v4/validator/keymanager"
)

// Wallet contains an in-memory, simulated wallet implementation.
type Wallet struct {
	InnerAccountsDir  string
	Directories       []string
	Files             map[string]map[string][]byte
	EncryptedSeedFile []byte
	AccountPasswords  map[string]string
	WalletPassword    string
	UnlockAccounts    bool
	lock              sync.RWMutex
	HasWriteFileError bool
}

// AccountNames --
func (w *Wallet) AccountNames() ([]string, error) {
	w.lock.RLock()
	defer w.lock.RUnlock()
	names := make([]string, 0)
	for name := range w.AccountPasswords {
		names = append(names, name)
	}
	return names, nil
}

// AccountsDir --
func (w *Wallet) AccountsDir() string {
	return w.InnerAccountsDir
}

// Exists --
func (w *Wallet) Exists() (bool, error) {
	return len(w.Directories) > 0, nil
}

// Password --
func (w *Wallet) Password() string {
	return w.WalletPassword
}

// WriteFileAtPath --
func (w *Wallet) WriteFileAtPath(_ context.Context, pathName, fileName string, data []byte) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.HasWriteFileError {
		// reset the flag to not contaminate other tests
		w.HasWriteFileError = false
		return errors.New("could not write keystore file for accounts")
	}
	if w.Files[pathName] == nil {
		w.Files[pathName] = make(map[string][]byte)
	}
	w.Files[pathName][fileName] = data
	return nil
}

// ReadFileAtPath --
func (w *Wallet) ReadFileAtPath(_ context.Context, pathName, fileName string) ([]byte, error) {
	w.lock.RLock()
	defer w.lock.RUnlock()
	for f, v := range w.Files[pathName] {
		if strings.Contains(fileName, f) {
			return v, nil
		}
	}
	return nil, errors.New("no files found")
}

// InitializeKeymanager --
func (_ *Wallet) InitializeKeymanager(_ context.Context, _ iface.InitKeymanagerConfig) (keymanager.IKeymanager, error) {
	return nil, nil
}

type MockValidator struct {
	Km               keymanager.IKeymanager
	proposerSettings *validatorserviceconfig.ProposerSettings
}

func (_ MockValidator) LogSyncCommitteeMessagesSubmitted() {}

func (_ MockValidator) Done() {
	panic("implement me")
}

func (_ MockValidator) WaitForChainStart(_ context.Context) error {
	panic("implement me")
}

func (_ MockValidator) WaitForSync(_ context.Context) error {
	panic("implement me")
}

func (_ MockValidator) WaitForActivation(_ context.Context, _ chan [][48]byte) error {
	panic("implement me")
}

func (_ MockValidator) CanonicalHeadSlot(_ context.Context) (primitives.Slot, error) {
	panic("implement me")
}

func (_ MockValidator) NextSlot() <-chan primitives.Slot {
	panic("implement me")
}

func (_ MockValidator) SlotDeadline(_ primitives.Slot) time.Time {
	panic("implement me")
}

func (_ MockValidator) LogValidatorGainsAndLosses(_ context.Context, _ primitives.Slot) error {
	panic("implement me")
}

func (_ MockValidator) UpdateDuties(_ context.Context, _ primitives.Slot) error {
	panic("implement me")
}

func (_ MockValidator) RolesAt(_ context.Context, _ primitives.Slot) (map[[48]byte][]iface2.ValidatorRole, error) {
	panic("implement me")
}

func (_ MockValidator) SubmitAttestation(_ context.Context, _ primitives.Slot, _ [48]byte) {
	panic("implement me")
}

func (_ MockValidator) ProposeBlock(_ context.Context, _ primitives.Slot, _ [48]byte) {
	panic("implement me")
}

func (_ MockValidator) SubmitAggregateAndProof(_ context.Context, _ primitives.Slot, _ [48]byte) {
	panic("implement me")
}

func (_ MockValidator) SubmitSyncCommitteeMessage(_ context.Context, _ primitives.Slot, _ [48]byte) {
	panic("implement me")
}

func (_ MockValidator) SubmitSignedContributionAndProof(_ context.Context, _ primitives.Slot, _ [48]byte) {
	panic("implement me")
}

func (_ MockValidator) LogAttestationsSubmitted() {
	panic("implement me")
}

func (_ MockValidator) UpdateDomainDataCaches(_ context.Context, _ primitives.Slot) {
	panic("implement me")
}

func (_ MockValidator) WaitForKeymanagerInitialization(_ context.Context) error {
	panic("implement me")
}

func (_ MockValidator) AllValidatorsAreExited(_ context.Context) (bool, error) {
	panic("implement me")
}

func (m MockValidator) Keymanager() (keymanager.IKeymanager, error) {
	return m.Km, nil
}

func (_ MockValidator) ReceiveBlocks(_ context.Context, _ chan<- error) {
	panic("implement me")
}

func (_ MockValidator) HandleKeyReload(_ context.Context, _ [][48]byte) (bool, error) {
	panic("implement me")
}

func (_ MockValidator) CheckDoppelGanger(_ context.Context) error {
	panic("implement me")
}

// HasProposerSettings for mocking
func (MockValidator) HasProposerSettings() bool {
	panic("implement me")
}

// PushProposerSettings for mocking
func (_ MockValidator) PushProposerSettings(_ context.Context, _ keymanager.IKeymanager, _ primitives.Slot, _ time.Time) error {
	panic("implement me")
}

// SetPubKeyToValidatorIndexMap for mocking
func (_ MockValidator) SetPubKeyToValidatorIndexMap(_ context.Context, _ keymanager.IKeymanager) error {
	panic("implement me")
}

// SignValidatorRegistrationRequest for mocking
func (_ MockValidator) SignValidatorRegistrationRequest(_ context.Context, _ iface2.SigningFunc, _ *ethpb.ValidatorRegistrationV1) (*ethpb.SignedValidatorRegistrationV1, error) {
	panic("implement me")
}

// ProposerSettings for mocking
func (m *MockValidator) ProposerSettings() *validatorserviceconfig.ProposerSettings {
	return m.proposerSettings
}

// SetProposerSettings for mocking
func (m *MockValidator) SetProposerSettings(settings *validatorserviceconfig.ProposerSettings) {
	m.proposerSettings = settings
}

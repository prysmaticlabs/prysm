package mock

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	iface2 "github.com/prysmaticlabs/prysm/validator/client/iface"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
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
	Km keymanager.IKeymanager
}

func (m MockValidator) Done() {
	panic("implement me")
}

func (m MockValidator) WaitForChainStart(ctx context.Context) error {
	panic("implement me")
}

func (m MockValidator) WaitForSync(ctx context.Context) error {
	panic("implement me")
}

func (m MockValidator) WaitForActivation(ctx context.Context, accountsChangedChan chan [][48]byte) error {
	panic("implement me")
}

func (m MockValidator) CanonicalHeadSlot(ctx context.Context) (types.Slot, error) {
	panic("implement me")
}

func (m MockValidator) NextSlot() <-chan types.Slot {
	panic("implement me")
}

func (m MockValidator) SlotDeadline(slot types.Slot) time.Time {
	panic("implement me")
}

func (m MockValidator) LogValidatorGainsAndLosses(ctx context.Context, slot types.Slot) error {
	panic("implement me")
}

func (m MockValidator) UpdateDuties(ctx context.Context, slot types.Slot) error {
	panic("implement me")
}

func (m MockValidator) RolesAt(ctx context.Context, slot types.Slot) (map[[48]byte][]iface2.ValidatorRole, error) {
	panic("implement me")
}

func (m MockValidator) SubmitAttestation(ctx context.Context, slot types.Slot, pubKey [48]byte) {
	panic("implement me")
}

func (m MockValidator) ProposeBlock(ctx context.Context, slot types.Slot, pubKey [48]byte) {
	panic("implement me")
}

func (m MockValidator) SubmitAggregateAndProof(ctx context.Context, slot types.Slot, pubKey [48]byte) {
	panic("implement me")
}

func (m MockValidator) SubmitSyncCommitteeMessage(ctx context.Context, slot types.Slot, pubKey [48]byte) {
	panic("implement me")
}

func (m MockValidator) SubmitSignedContributionAndProof(ctx context.Context, slot types.Slot, pubKey [48]byte) {
	panic("implement me")
}

func (m MockValidator) LogAttestationsSubmitted() {
	panic("implement me")
}

func (m MockValidator) LogNextDutyTimeLeft(slot types.Slot) error {
	panic("implement me")
}

func (m MockValidator) UpdateDomainDataCaches(ctx context.Context, slot types.Slot) {
	panic("implement me")
}

func (m MockValidator) WaitForKeymanagerInitialization(ctx context.Context) error {
	panic("implement me")
}

func (m MockValidator) AllValidatorsAreExited(ctx context.Context) (bool, error) {
	panic("implement me")
}

func (m MockValidator) Keymanager() (keymanager.IKeymanager, error) {
	return m.Km, nil
}

func (m MockValidator) ReceiveBlocks(ctx context.Context, connectionErrorChannel chan<- error) {
	panic("implement me")
}

func (m MockValidator) HandleKeyReload(ctx context.Context, newKeys [][48]byte) (bool, error) {
	panic("implement me")
}

func (m MockValidator) CheckDoppelGanger(ctx context.Context) error {
	panic("implement me")
}

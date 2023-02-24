package mock

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/v3/async/event"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpbservice "github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	validatorpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
)

// MockKeymanager --
type MockKeymanager struct {
	PublicKeys             [][fieldparams.BLSPubkeyLength]byte
	ReloadPublicKeysChan   chan [][fieldparams.BLSPubkeyLength]byte
	ReloadPublicKeysCalled bool
	accountsChangedFeed    *event.Feed
}

func NewMock() MockKeymanager {
	return MockKeymanager{
		accountsChangedFeed:  new(event.Feed),
		ReloadPublicKeysChan: make(chan [][fieldparams.BLSPubkeyLength]byte, 1),
	}
}

// FetchValidatingPublicKeys --
func (m *MockKeymanager) FetchValidatingPublicKeys(context.Context) ([][fieldparams.BLSPubkeyLength]byte, error) {
	return m.PublicKeys, nil
}

// Sign --
func (*MockKeymanager) Sign(_ context.Context, s *validatorpb.SignRequest) (bls.Signature, error) {
	key, err := bls.RandKey()
	if err != nil {
		return nil, err
	}
	st, _ := util.DeterministicGenesisState(nil, 1)
	e := slots.ToEpoch(st.Slot())
	byteValue, err := signing.ComputeDomainAndSign(st, e, s.SigningSlot, bytesutil.ToBytes4(s.SignatureDomain), key)
	if err != nil {
		return nil, err
	}
	return bls.SignatureFromBytes(byteValue)
}

// SubscribeAccountChanges --
func (m *MockKeymanager) SubscribeAccountChanges(chan [][fieldparams.BLSPubkeyLength]byte) event.Subscription {
	return m.accountsChangedFeed.Subscribe(m.ReloadPublicKeysChan)
}

// ReloadPublicKeys --
func (m *MockKeymanager) ReloadPublicKeys(context.Context) ([][fieldparams.BLSPubkeyLength]byte, error) {
	m.ReloadPublicKeysCalled = true
	m.ReloadPublicKeysChan <- m.PublicKeys
	return m.PublicKeys, nil
}

// ExtractKeystores --
func (*MockKeymanager) ExtractKeystores(
	_ context.Context, _ []bls.PublicKey, _ string,
) ([]*keymanager.Keystore, error) {
	return nil, errors.New("extracting keys not supported for a remote keymanager")
}

// ListKeymanagerAccounts --
func (*MockKeymanager) ListKeymanagerAccounts(
	context.Context, keymanager.ListKeymanagerAccountConfig) error {
	return nil
}

func (*MockKeymanager) DeleteKeystores(context.Context, [][]byte,
) ([]*ethpbservice.DeletedKeystoreStatus, error) {
	return nil, nil
}

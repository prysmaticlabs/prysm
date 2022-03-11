package mock

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/async/event"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
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
func (*MockKeymanager) Sign(context.Context, *validatorpb.SignRequest) (bls.Signature, error) {
	panic("implement me")
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
	ctx context.Context, publicKeys []bls.PublicKey, password string,
) ([]*keymanager.Keystore, error) {
	return nil, errors.New("extracting keys not supported for a remote keymanager")
}

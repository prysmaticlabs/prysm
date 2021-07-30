package remote

import (
	"context"

	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/event"
)

// MockKeymanager --
type MockKeymanager struct {
	PublicKeys             [][48]byte
	ReloadPublicKeysChan   chan [][48]byte
	ReloadPublicKeysCalled bool
	accountsChangedFeed    *event.Feed
}

func NewMock() MockKeymanager {
	return MockKeymanager{
		accountsChangedFeed:  new(event.Feed),
		ReloadPublicKeysChan: make(chan [][48]byte, 1),
	}
}

// FetchValidatingPublicKeys --
func (m *MockKeymanager) FetchValidatingPublicKeys(context.Context) ([][48]byte, error) {
	return m.PublicKeys, nil
}

// Sign --
func (*MockKeymanager) Sign(context.Context, *validatorpb.SignRequest) (bls.Signature, error) {
	panic("implement me")
}

// SubscribeAccountChanges --
func (m *MockKeymanager) SubscribeAccountChanges(chan [][48]byte) event.Subscription {
	return m.accountsChangedFeed.Subscribe(m.ReloadPublicKeysChan)
}

// ReloadPublicKeys --
func (m *MockKeymanager) ReloadPublicKeys(context.Context) ([][48]byte, error) {
	m.ReloadPublicKeysCalled = true
	m.ReloadPublicKeysChan <- m.PublicKeys
	return m.PublicKeys, nil
}

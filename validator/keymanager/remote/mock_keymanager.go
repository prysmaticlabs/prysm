package remote

import (
	"context"

	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/event"
)

type MockKeymanager struct {
	PublicKeys             [][48]byte
	ReloadPublicKeysChan   chan [][48]byte
	ReloadPublicKeysCalled bool
	accountsChangedFeed    *event.Feed
}

func New() MockKeymanager {
	return MockKeymanager{
		accountsChangedFeed:  new(event.Feed),
		ReloadPublicKeysChan: make(chan [][48]byte, 1),
	}
}

func (m *MockKeymanager) FetchValidatingPublicKeys(context.Context) ([][48]byte, error) {
	return m.PublicKeys, nil
}

func (*MockKeymanager) Sign(context.Context, *validatorpb.SignRequest) (bls.Signature, error) {
	panic("implement me")
}

func (m *MockKeymanager) SubscribeAccountChanges(chan [][48]byte) event.Subscription {
	return m.accountsChangedFeed.Subscribe(m.ReloadPublicKeysChan)
}

func (m *MockKeymanager) ReloadPublicKeys(context.Context) ([][48]byte, error) {
	m.ReloadPublicKeysCalled = true
	m.ReloadPublicKeysChan <- m.PublicKeys
	return m.PublicKeys, nil
}

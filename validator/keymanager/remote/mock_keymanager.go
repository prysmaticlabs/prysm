package remote

import (
	"context"

	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/event"
)

// MockKeymanager --
type MockKeymanager struct {
	PublicKeys [][48]byte
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
func (*MockKeymanager) SubscribeAccountChanges(chan [][48]byte) event.Subscription {
	panic("implement me")
}

// ReloadPublicKeys --
func (m *MockKeymanager) ReloadPublicKeys(context.Context) ([][48]byte, error) {
	return m.PublicKeys, nil
}

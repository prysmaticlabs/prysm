package remote

import (
	"context"

	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/event"
)

type MockKeymanager struct {
	PublicKeys [][48]byte
}

func (m *MockKeymanager) FetchValidatingPublicKeys(context.Context) ([][48]byte, error) {
	return m.PublicKeys, nil
}

func (*MockKeymanager) Sign(context.Context, *validatorpb.SignRequest) (bls.Signature, error) {
	panic("implement me")
}

func (*MockKeymanager) SubscribeAccountChanges(chan [][48]byte) event.Subscription {
	panic("implement me")
}

func (m *MockKeymanager) ReloadPublicKeys(context.Context) ([][48]byte, error) {
	return m.PublicKeys, nil
}

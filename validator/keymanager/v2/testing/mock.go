package testing

import (
	"context"
	"errors"

	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// MockKeymanager --
type MockKeymanager struct {
	ConfigFileContents  []byte
	PublicKeys          [][48]byte
	PubkeystoSecretKeys map[[48]byte]bls.SecretKey
}

// CreateAccount --
func (m *MockKeymanager) CreateAccount(ctx context.Context, password string) (string, error) {
	return "", nil
}

// MarshalConfigFile --
func (m *MockKeymanager) MarshalConfigFile(ctx context.Context) ([]byte, error) {
	return m.ConfigFileContents, nil
}

// FetchValidatingPublicKeys --
func (m *MockKeymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	return m.PublicKeys, nil
}

// Sign --
func (m *MockKeymanager) Sign(ctx context.Context, req *validatorpb.SignRequest) (bls.Signature, error) {
	pubKey := bytesutil.ToBytes48(req.PublicKey)
	secretKey, ok := m.PubkeystoSecretKeys[pubKey]
	if !ok {
		return nil, errors.New("no secret key found")
	}
	return secretKey.Sign(req.Data), nil
}

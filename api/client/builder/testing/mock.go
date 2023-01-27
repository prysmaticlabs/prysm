package testing

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/api/client/builder"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// MockClient is a mock implementation of BuilderClient.
type MockClient struct {
	RegisteredVals map[[48]byte]bool
}

// NewClient creates a new, correctly initialized mock.
func NewClient() MockClient {
	return MockClient{RegisteredVals: map[[48]byte]bool{}}
}

// NodeURL --
func (MockClient) NodeURL() string {
	return ""
}

// GetHeader --
func (MockClient) GetHeader(_ context.Context, _ primitives.Slot, _ [32]byte, _ [48]byte) (builder.SignedBid, error) {
	return nil, nil
}

// RegisterValidator --
func (m MockClient) RegisterValidator(_ context.Context, svr []*ethpb.SignedValidatorRegistrationV1) error {
	for _, r := range svr {
		b := bytesutil.ToBytes48(r.Message.Pubkey)
		m.RegisteredVals[b] = true
	}
	return nil
}

// SubmitBlindedBlock --
func (MockClient) SubmitBlindedBlock(_ context.Context, _ interfaces.SignedBeaconBlock) (interfaces.ExecutionData, error) {
	return nil, nil
}

// Status --
func (MockClient) Status(_ context.Context) error {
	return nil
}

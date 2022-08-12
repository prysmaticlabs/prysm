package testing

import (
	"context"

	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

type MockClient struct {
	RegisteredVals map[[48]byte]bool
}

func NewClient() MockClient {
	return MockClient{RegisteredVals: map[[48]byte]bool{}}
}

func (MockClient) NodeURL() string {
	return ""
}

func (MockClient) GetHeader(_ context.Context, _ types.Slot, _ [32]byte, _ [48]byte) (*ethpb.SignedBuilderBid, error) {
	return nil, nil
}

func (m MockClient) RegisterValidator(_ context.Context, svr []*ethpb.SignedValidatorRegistrationV1) error {
	for _, r := range svr {
		b := bytesutil.ToBytes48(r.Message.Pubkey)
		m.RegisteredVals[b] = true
	}
	return nil
}

func (MockClient) SubmitBlindedBlock(_ context.Context, _ *ethpb.SignedBlindedBeaconBlockBellatrix) (*v1.ExecutionPayload, error) {
	return nil, nil
}

func (MockClient) Status(_ context.Context) error {
	return nil
}

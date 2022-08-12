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

func (m MockClient) NodeURL() string {
	return ""
}

func (m MockClient) GetHeader(ctx context.Context, slot types.Slot, parentHash [32]byte, pubkey [48]byte) (*ethpb.SignedBuilderBid, error) {
	return nil, nil
}

func (m MockClient) RegisterValidator(ctx context.Context, svr []*ethpb.SignedValidatorRegistrationV1) error {
	for _, r := range svr {
		b := bytesutil.ToBytes48(r.Message.Pubkey)
		m.RegisteredVals[b] = true
	}
	return nil
}

func (m MockClient) SubmitBlindedBlock(ctx context.Context, sb *ethpb.SignedBlindedBeaconBlockBellatrix) (*v1.ExecutionPayload, error) {
	return nil, nil
}

func (m MockClient) Status(ctx context.Context) error {
	return nil
}

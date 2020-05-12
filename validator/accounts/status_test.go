package accounts

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/validator/internal"
)

func TestFetchAccountStatuses_OK(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	const numBatches = 5
	pubkeys := make([][]byte, MaxRequestKeys*numBatches)
	for i := 0; i < MaxRequestKeys*numBatches; i++ {
		pubkeyBytes := bytesutil.ToBytes48([]byte{byte(i)})
		pubkeys[i] = pubkeyBytes[:]
	}
	mockClient := internal.NewMockBeaconNodeValidatorClient(ctrl)
	for i := 0; i+MaxRequestKeys <= len(pubkeys); i += MaxRequestKeys {
		mockClient.EXPECT().MultipleValidatorStatus(
			gomock.Any(),
			&ethpb.MultipleValidatorStatusRequest{
				PublicKeys: pubkeys[i : i+MaxRequestKeys],
			},
		)
	}
	_, err := FetchAccountStatuses(ctx, mockClient, pubkeys)
	if err != nil {
		t.Fatalf("FetchAccountStatuses failed with error: %v.", err)
	}
}

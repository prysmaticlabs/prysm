package accounts

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/validator/internal"
)

func TestFetchAccountStatuses_OK(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := internal.NewMockBeaconNodeValidatorClient(ctrl)
	key, err := keystore.NewKey()
	if err != nil {
		t.Fatal("Failed to generate new key.")
	}
	keyPairs := map[string]*keystore.Key{"": key}
	mockClient.EXPECT().ValidatorStatus(
		gomock.Any(),
		&ethpb.ValidatorStatusRequest{
			PublicKey: key.PublicKey.Marshal(),
		},
	)
	_, err = FetchAccountStatuses(ctx, mockClient, keyPairs)
	if err != nil {
		t.Fatalf("FetchAccountStatuses failed with error: %v", err)
	}
}

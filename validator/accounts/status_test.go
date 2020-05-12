package accounts

import (
	"context"
	"encoding/hex"
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
	const numBatches = 5
	keyMap := make(map[string]*keystore.Key)
	for i := 0; i < MaxRequestKeys*numBatches; i++ {
		newKey, err := keystore.NewKey()
		if err != nil {
			t.Fatal("Failed to generate new key.")
		}
		keyMap[hex.EncodeToString(newKey.PublicKey.Marshal())] = newKey
	}
	pubkeys := ExtractPublicKeys(keyMap)
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

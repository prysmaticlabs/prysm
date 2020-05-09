package accounts

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/validator/internal"
)

func TestFetchAccountStatuses_OK(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := internal.NewMockBeaconNodeValidatorClient(ctrl)
	keyPairs := make(map[string]*keystore.Key)
	_, err := FetchAccountStatuses(ctx, mockClient, keyPairs)
	if err != nil {
		t.Fatalf("FetchAccountStatuses failed with error: %v", err)
	}
}

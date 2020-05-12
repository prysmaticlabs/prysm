package accounts

import (
	"context"
	"sort"
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

func TestMergeStatuses_OK(t *testing.T) {
	const numBatches = 3
	const numStatusTypes = int(ethpb.ValidatorStatus_EXITED) + 1
	allStatuses := make([][]ValidatorStatusMetadata, numBatches)
	for i := 0; i < numBatches; i++ {
		statuses := make([]ValidatorStatusMetadata, MaxRequestKeys)
		for j := 0; j < MaxRequestKeys; j++ {
			statuses[j] = ValidatorStatusMetadata{
				Metadata: &ethpb.ValidatorStatusResponse{
					Status: ethpb.ValidatorStatus(j % numStatusTypes),
				},
			}
		}
		sort.Slice(statuses, func(k, l int) bool {
			return statuses[k].Metadata.Status < statuses[l].Metadata.Status
		})
		allStatuses[i] = statuses
	}
	sortedStatuses := MergeStatuses(allStatuses)
	isSorted := sort.SliceIsSorted(sortedStatuses, func(i, j int) bool {
		return sortedStatuses[i].Metadata.Status < sortedStatuses[j].Metadata.Status
	})
	if !isSorted {
		t.Fatalf("Merge Status failed. Statuses are not sorted.")
	}
}

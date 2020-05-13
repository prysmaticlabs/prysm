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
	const NumBatches = 5
	pubkeys := make([][]byte, MaxRequestKeys*NumBatches)
	for i := 0; i < MaxRequestKeys*NumBatches; i++ {
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
	allStatuses, err := FetchAccountStatuses(ctx, mockClient, pubkeys)
	if err != nil {
		t.Fatalf("FetchAccountStatuses failed with error: %v.", err)
	}
	if len(allStatuses) != NumBatches {
		t.Fatalf("Want %d status arrays. Recieved %d", NumBatches, len(allStatuses))
	}
}

func TestMergeStatuses_OK(t *testing.T) {
	const NumBatches = 5
	const NumStatusTypes = int(ethpb.ValidatorStatus_EXITED) + 1
	allStatuses := make([][]ValidatorStatusMetadata, NumBatches)
	for i := 0; i < NumBatches; i++ {
		statuses := make([]ValidatorStatusMetadata, MaxRequestKeys)
		for j := 0; j < MaxRequestKeys; j++ {
			statuses[j] = ValidatorStatusMetadata{
				Metadata: &ethpb.ValidatorStatusResponse{
					Status: ethpb.ValidatorStatus(j % NumStatusTypes),
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

package beaconclient

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestService_RequestHistoricalAttestations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)

	bs := Service{
		beaconClient: client,
	}

	numAtts := 1000
	wanted := make([]*ethpb.IndexedAttestation, numAtts)
	for i := 0; i < numAtts; i++ {
		wanted[i] = &ethpb.IndexedAttestation{
			AttestingIndices: []uint64{1, 2, 3},
			Data: &ethpb.AttestationData{
				Slot: uint64(i),
			},
		}
	}

	client.EXPECT().ListIndexedAttestations(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, nil)

	// We override the page size in the requests to 100 so we will
	// obtain 10 pages of indexed attestations from the server.
	cfg := params.BeaconConfig()
	cfg.DefaultPageSize = 100
	params.OverrideBeaconConfig(cfg)

	// We request attestations for epoch 0.
	res, err := bs.RequestHistoricalAttestations(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(res, wanted) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

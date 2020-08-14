package beaconclient

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestService_RequestHistoricalAttestations(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := testDB.SetupSlasherDB(t, false)
	client := mock.NewMockBeaconChainClient(ctrl)

	bs := Service{
		beaconClient: client,
		slasherDB:    db,
	}

	numAtts := 1000
	wanted := make([]*ethpb.IndexedAttestation, numAtts)
	for i := 0; i < numAtts; i++ {
		wanted[i] = &ethpb.IndexedAttestation{
			AttestingIndices: []uint64{1, 2, 3},
			Data: &ethpb.AttestationData{
				Slot: uint64(i),
				Target: &ethpb.Checkpoint{
					Epoch: 1,
					Root:  make([]byte, 32),
				},
			},
		}
	}

	// We override the page size in the requests to 100 so we will
	// obtain 10 pages of indexed attestations from the server.
	numPages := 100
	perPage := numAtts / numPages
	cfg := params.BeaconConfig()
	cfg.DefaultPageSize = perPage
	params.OverrideBeaconConfig(cfg)

	// We expect there to be numPages calls to ListIndexedAttestations
	// to retrieve all attestations for epoch 0.
	for i := 0; i < numAtts; i += perPage {
		if i+perPage >= numAtts {
			client.EXPECT().ListIndexedAttestations(
				gomock.Any(),
				gomock.Any(),
			).Return(&ethpb.ListIndexedAttestationsResponse{
				IndexedAttestations: wanted[i:],
				NextPageToken:       "",
				TotalSize:           int32(numAtts),
			}, nil)
		} else {
			client.EXPECT().ListIndexedAttestations(
				gomock.Any(),
				gomock.Any(),
			).Return(&ethpb.ListIndexedAttestationsResponse{
				IndexedAttestations: wanted[i : i+perPage],
				NextPageToken:       strconv.Itoa(i + 1),
				TotalSize:           int32(numAtts),
			}, nil)
		}
	}

	// We request attestations for epoch 0.
	res, err := bs.RequestHistoricalAttestations(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res, wanted) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
	require.LogsContain(t, hook, "Retrieved 100/1000 indexed attestations for epoch 0")
	require.LogsContain(t, hook, "Retrieved 500/1000 indexed attestations for epoch 0")
	require.LogsContain(t, hook, "Retrieved 1000/1000 indexed attestations for epoch 0")
}

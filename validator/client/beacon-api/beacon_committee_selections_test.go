package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/validator"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api/mock"
	test_helpers "github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api/test-helpers"
)

func TestGetAggregatedSelections(t *testing.T) {
	testcases := []struct {
		name                 string
		req                  []validator.BeaconCommitteeSelection
		res                  []validator.BeaconCommitteeSelection
		endpointError        error
		expectedErrorMessage string
	}{
		{
			name: "valid",
			req: []validator.BeaconCommitteeSelection{
				{
					SelectionProof: test_helpers.FillByteSlice(96, 82),
					Slot:           75,
					ValidatorIndex: 76,
				},
			},
			res: []validator.BeaconCommitteeSelection{
				{
					SelectionProof: test_helpers.FillByteSlice(96, 100),
					Slot:           75,
					ValidatorIndex: 76,
				},
			},
		},
		{
			name: "endpoint error",
			req: []validator.BeaconCommitteeSelection{
				{
					SelectionProof: test_helpers.FillByteSlice(96, 82),
					Slot:           75,
					ValidatorIndex: 76,
				},
			},
			endpointError:        errors.New("bad request"),
			expectedErrorMessage: "bad request",
		},
		{
			name: "no response error",
			req: []validator.BeaconCommitteeSelection{
				{
					SelectionProof: test_helpers.FillByteSlice(96, 82),
					Slot:           75,
					ValidatorIndex: 76,
				},
			},
			expectedErrorMessage: "no aggregated selection returned",
		},
		{
			name: "mismatch response",
			req: []validator.BeaconCommitteeSelection{
				{
					SelectionProof: test_helpers.FillByteSlice(96, 82),
					Slot:           75,
					ValidatorIndex: 76,
				},
				{
					SelectionProof: test_helpers.FillByteSlice(96, 102),
					Slot:           75,
					ValidatorIndex: 79,
				},
			},
			res: []validator.BeaconCommitteeSelection{
				{
					SelectionProof: test_helpers.FillByteSlice(96, 100),
					Slot:           75,
					ValidatorIndex: 76,
				},
			},
			expectedErrorMessage: "mismatching number of selections",
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			jsonRestHandler := mock.NewMockJsonRestHandler(ctrl)

			reqBody, err := json.Marshal(test.req)
			require.NoError(t, err)

			ctx := context.Background()
			jsonRestHandler.EXPECT().Post(
				ctx,
				"/eth/v1/validator/beacon_committee_selections",
				nil,
				bytes.NewBuffer(reqBody),
				&aggregatedSelectionResponse{},
			).SetArg(
				4,
				aggregatedSelectionResponse{Data: test.res},
			).Return(
				nil,
				test.endpointError,
			).Times(1)

			validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
			res, err := validatorClient.GetAggregatedSelections(ctx, test.req)
			if test.expectedErrorMessage != "" {
				require.ErrorContains(t, test.expectedErrorMessage, err)
				return
			}

			require.NoError(t, err)
			require.DeepEqual(t, test.res, res)
		})
	}
}

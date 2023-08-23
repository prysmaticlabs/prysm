package validator

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	chainMock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestGetValidatorCount(t *testing.T) {
	st, _ := util.DeterministicGenesisState(t, 10)
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch
	validators := []*eth.Validator{
		// Pending initialized.
		{
			ActivationEpoch:            farFutureEpoch,
			ActivationEligibilityEpoch: farFutureEpoch,
			ExitEpoch:                  farFutureEpoch,
			WithdrawableEpoch:          farFutureEpoch,
		},
		// Pending queued.
		{
			ActivationEpoch:            10,
			ActivationEligibilityEpoch: 4,
			ExitEpoch:                  farFutureEpoch,
			WithdrawableEpoch:          farFutureEpoch,
		},
		// Active ongoing.
		{
			ActivationEpoch: 0,
			ExitEpoch:       farFutureEpoch,
		},
		// Active slashed.
		{
			ActivationEpoch:   0,
			ExitEpoch:         30,
			Slashed:           true,
			WithdrawableEpoch: 50,
		},
		// Active exiting.
		{
			ActivationEpoch:   0,
			ExitEpoch:         30,
			Slashed:           false,
			WithdrawableEpoch: 50,
		},
		// Exit slashed (at epoch 35).
		{
			ActivationEpoch:   3,
			ExitEpoch:         30,
			WithdrawableEpoch: 50,
			Slashed:           true,
		},
		// Exit unslashed (at epoch 35).
		{
			ActivationEpoch:   3,
			ExitEpoch:         30,
			WithdrawableEpoch: 50,
			Slashed:           false,
		},
		// Withdrawable (at epoch 45).
		{
			ActivationEpoch:   3,
			ExitEpoch:         30,
			WithdrawableEpoch: 40,
			EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
			Slashed:           false,
		},
		// Withdrawal done (at epoch 45).
		{
			ActivationEpoch:   3,
			ExitEpoch:         30,
			WithdrawableEpoch: 40,
			EffectiveBalance:  0,
			Slashed:           false,
		},
	}
	for _, validator := range validators {
		require.NoError(t, st.AppendValidator(validator))
		require.NoError(t, st.AppendBalance(params.BeaconConfig().MaxEffectiveBalance))
	}

	tests := []struct {
		name             string
		stateID          string
		status           string
		currentEpoch     int
		expectedResponse ValidatorCountResponse
	}{
		{
			name:    "Head count active validators",
			stateID: "head",
			status:  "active",
			expectedResponse: ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: &ValidatorCount{
					ValidatorCount: "13",
				},
			},
		},
		{
			name:    "Head count active ongoing validators",
			stateID: "head",
			status:  "active_ongoing",
			expectedResponse: ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: &ValidatorCount{
					ValidatorCount: "11",
				},
			},
		},
		{
			name:    "Head count active exiting validators",
			stateID: "head",
			status:  "active_exiting",
			expectedResponse: ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: &ValidatorCount{
					ValidatorCount: "1",
				},
			},
		},
		{
			name:    "Head count active slashed validators",
			stateID: "head",
			status:  "active_slashed",
			expectedResponse: ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: &ValidatorCount{
					ValidatorCount: "1",
				},
			},
		},
		{
			name:    "Head count pending validators",
			stateID: "head",
			status:  "pending",
			expectedResponse: ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: &ValidatorCount{
					ValidatorCount: "6",
				},
			},
		},
		{
			name:    "Head count pending initialized validators",
			stateID: "head",
			status:  "pending_initialized",
			expectedResponse: ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: &ValidatorCount{
					ValidatorCount: "1",
				},
			},
		},
		{
			name:    "Head count pending queued validators",
			stateID: "head",
			status:  "pending_queued",
			expectedResponse: ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: &ValidatorCount{
					ValidatorCount: "5",
				},
			},
		},
		{
			name:         "Head count exited validators",
			stateID:      "head",
			status:       "exited",
			currentEpoch: 35,
			expectedResponse: ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: &ValidatorCount{
					ValidatorCount: "6",
				},
			},
		},
		{
			name:         "Head count exited slashed validators",
			stateID:      "head",
			status:       "exited_slashed",
			currentEpoch: 35,
			expectedResponse: ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: &ValidatorCount{
					ValidatorCount: "2",
				},
			},
		},
		{
			name:         "Head count exited unslashed validators",
			stateID:      "head",
			status:       "exited_unslashed",
			currentEpoch: 35,
			expectedResponse: ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: &ValidatorCount{
					ValidatorCount: "4",
				},
			},
		},
		{
			name:         "Head count withdrawal validators",
			stateID:      "head",
			status:       "withdrawal",
			currentEpoch: 45,
			expectedResponse: ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: &ValidatorCount{
					ValidatorCount: "2",
				},
			},
		},
		{
			name:         "Head count withdrawal possible validators",
			stateID:      "head",
			status:       "withdrawal_possible",
			currentEpoch: 45,
			expectedResponse: ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: &ValidatorCount{
					ValidatorCount: "1",
				},
			},
		},
		{
			name:         "Head count withdrawal done validators",
			stateID:      "head",
			status:       "withdrawal_done",
			currentEpoch: 45,
			expectedResponse: ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: &ValidatorCount{
					ValidatorCount: "1",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chainService := &chainMock.ChainService{Optimistic: false, FinalizedRoots: make(map[[32]byte]bool)}
			blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
			require.NoError(t, err)
			chainService.FinalizedRoots[blockRoot] = true
			require.NoError(t, st.SetSlot(params.BeaconConfig().SlotsPerEpoch*primitives.Slot(test.currentEpoch)))

			server := &Server{
				OptimisticModeFetcher: chainService,
				FinalizationFetcher:   chainService,
				Stater: &testutil.MockStater{
					BeaconState: st,
				},
			}

			testRouter := mux.NewRouter()
			testRouter.HandleFunc("/eth/v1/beacon/states/{state_id}/validator_count", server.GetValidatorCount)
			s := httptest.NewServer(testRouter)
			defer s.Close()

			resp, err := http.Get(s.URL + fmt.Sprintf("/eth/v1/beacon/states/%s/validator_count?status=%s",
				test.stateID, test.status))
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			t.Log(string(body))

			var count ValidatorCountResponse
			err = json.Unmarshal(body, &count)
			require.NoError(t, err)
			require.DeepEqual(t, test.expectedResponse, count)
		})
	}
}

package beaconapi_evaluators

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/helpers"
	"google.golang.org/grpc"
)

func withCompareAttesterDuties(beaconNodeIdx int, conn *grpc.ClientConn) error {
	type attesterDutyJson struct {
		Pubkey                  string `json:"pubkey" hex:"true"`
		ValidatorIndex          string `json:"validator_index"`
		CommitteeIndex          string `json:"committee_index"`
		CommitteeLength         string `json:"committee_length"`
		CommitteesAtSlot        string `json:"committees_at_slot"`
		ValidatorCommitteeIndex string `json:"validator_committee_index"`
		Slot                    string `json:"slot"`
	}
	type attesterDutiesResponseJson struct {
		DependentRoot string              `json:"dependent_root" hex:"true"`
		Data          []*attesterDutyJson `json:"data"`
	}
	ctx := context.Background()
	validatorClient := service.NewBeaconValidatorClient(conn)
	resp, err := validatorClient.GetAttesterDuties(ctx, &ethpbv1.AttesterDutiesRequest{
		Epoch: helpers.AltairE2EForkEpoch,
		Index: []types.ValidatorIndex{0},
	})
	if err != nil {
		return err
	}
	// We post a top-level array, not an object, as per the spec.
	reqJSON := []string{"0"}
	respJSON := &attesterDutiesResponseJson{}
	if err := doMiddlewareJSONPostRequestV1(
		"/validator/duties/attester/"+strconv.Itoa(helpers.AltairE2EForkEpoch),
		beaconNodeIdx,
		reqJSON,
		respJSON,
	); err != nil {
		return err
	}
	if respJSON.DependentRoot != hexutil.Encode(resp.DependentRoot) {
		return buildFieldError("DependentRoot", string(resp.DependentRoot), respJSON.DependentRoot)
	}
	if len(respJSON.Data) != len(resp.Data) {
		return fmt.Errorf(
			"API Middleware number of duties %d does not match gRPC %d",
			len(respJSON.Data),
			len(resp.Data),
		)
	}
	return nil
}

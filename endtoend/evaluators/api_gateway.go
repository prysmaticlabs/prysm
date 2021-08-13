package evaluators

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/protobuf/ptypes/empty"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/policies"
	e2etypes "github.com/prysmaticlabs/prysm/endtoend/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"google.golang.org/grpc"
)

// APIGatewayV1Alpha1VerifyIntegrity of our API gateway for the Prysm v1alpha1 API.
// This ensures our gRPC HTTP gateway returns and processes the same data _for the same endpoints_
// as using a gRPC connection to interact with the API. Running this in end-to-end tests helps us
// ensure parity between our HTTP gateway for our API and gRPC never breaks.
// This evaluator checks a few request/response trips for both GET and POST requests.
var APIGatewayV1Alpha1VerifyIntegrity = e2etypes.Evaluator{
	Name:       "api_gateway_v1alpha1_verify_integrity_epoch_%d",
	Policy:     policies.OnEpoch(1),
	Evaluation: apiGatewayV1Alpha1Verify,
}

type chainHeadResponse struct {
	HeadSlot uint64
}

func apiGatewayV1Alpha1Verify(conns ...*grpc.ClientConn) error {
	for idx, conn := range conns {
		beaconClient := ethpb.NewBeaconChainClient(conn)
		ctx := context.Background()
		resp, err := beaconClient.GetChainHead(ctx, &empty.Empty{})
		if err != nil {
			return err
		}
		_ = resp
		basePath := fmt.Sprintf("http://localhost:%d/eth/v1alpha1", e2e.TestParams.BeaconNodeRPCPort+idx+40)
		apiresp, err := http.Get(
			basePath + "/beacon/chainhead",
		)
		if err != nil {
			return err
		}
		httpChainHeadResp := &chainHeadResponse{}
		if err = json.NewDecoder(apiresp.Body).Decode(&httpChainHeadResp); err != nil {
			return err
		}
		if httpChainHeadResp.HeadSlot != uint64(resp.HeadSlot) {
			return fmt.Errorf("HTTP gateway chainhead %v does not match gRPC chainhead %v", httpChainHeadResp, resp)
		}
	}
	return nil
}

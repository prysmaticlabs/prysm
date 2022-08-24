package beaconapi_evaluators

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"google.golang.org/grpc"
)

// "/eth/v1/beacon/blocks/{block_id}"
func withCompareBeaconBlocks(beaconNodeIdx int, conn *grpc.ClientConn) error {
	ctx := context.Background()
	beaconClient := service.NewBeaconChainClient(conn)
	genesisData, err := beaconClient.GetGenesis(ctx, &empty.Empty{})
	if err != nil {
		return err
	}
	currentEpoch := slots.EpochsSinceGenesis(genesisData.Data.GenesisTime.AsTime())
	if currentEpoch < params.BeaconConfig().AltairForkEpoch {
		resp, err := beaconClient.GetBlock(ctx, &ethpbv1.BlockRequest{
			BlockId: []byte("head"),
		})
		if err != nil {
			return err
		}
		respJSON := &apimiddleware.BlockResponseJson{}
		if err := doMiddlewareJSONGetRequest(
			v1MiddlewarePathTemplate,
			"/beacon/blocks/head",
			beaconNodeIdx,
			respJSON,
		); err != nil {
			return err
		}
		if hexutil.Encode(resp.Data.Signature) != respJSON.Data.Signature {
			return fmt.Errorf("API Middleware block signature  %s does not match gRPC block signature %s",
				respJSON.Data.Signature,
				hexutil.Encode(resp.Data.Signature))
		}
	} else {
		resp, err := beaconClient.GetBlockV2(ctx, &ethpbv2.BlockRequestV2{
			BlockId: []byte("head"),
		})
		if err != nil {
			return err
		}
		respJSON := &apimiddleware.BlockResponseJson{}
		if err := doMiddlewareJSONGetRequest(
			v2MiddlewarePathTemplate,
			"/beacon/blocks/head",
			beaconNodeIdx,
			respJSON,
		); err != nil {
			return err
		}
		if hexutil.Encode(resp.Data.Signature) != respJSON.Data.Signature {
			return fmt.Errorf("API Middleware block signature  %s does not match gRPC block signature %s",
				respJSON.Data.Signature,
				hexutil.Encode(resp.Data.Signature))
		}
	}

	return nil
}

// eth/v1/beacon/states/{state_id}/validators
func withCompareValidatorsEth(beaconNodeIdx int, conn *grpc.ClientConn) error {
	type stateValidatorsResponseJson struct {
		Data []*validatorContainerJson `json:"data"`
	}
	ctx := context.Background()
	beaconClient := service.NewBeaconChainClient(conn)
	resp, err := beaconClient.ListValidators(ctx, &ethpbv1.StateValidatorsRequest{
		StateId: []byte("head"),
		Status:  []ethpbv1.ValidatorStatus{ethpbv1.ValidatorStatus_EXITED},
	})
	if err != nil {
		return err
	}
	respJSON := &stateValidatorsResponseJson{}
	if err := doMiddlewareJSONGetRequest(
		v1MiddlewarePathTemplate,
		"/beacon/states/head/validators?status=exited",
		beaconNodeIdx,
		respJSON,
	); err != nil {
		return err
	}
	if len(respJSON.Data) != len(resp.Data) {
		return fmt.Errorf(
			"API Middleware number of validators %d does not match gRPC %d",
			len(respJSON.Data),
			len(resp.Data),
		)
	}
	resp, err = beaconClient.ListValidators(ctx, &ethpbv1.StateValidatorsRequest{
		StateId: []byte("head"),
		Id:      [][]byte{[]byte("100"), []byte("200")},
	})
	if err != nil {
		return err
	}
	if err := doMiddlewareJSONGetRequest(
		v1MiddlewarePathTemplate,
		"/beacon/states/head/validators?id=100&id=200",
		beaconNodeIdx,
		respJSON,
	); err != nil {
		return err
	}
	if len(respJSON.Data) != len(resp.Data) {
		return fmt.Errorf(
			"API Middleware number of validators %d does not match gRPC %d",
			len(respJSON.Data),
			len(resp.Data),
		)
	}
	if err = assertValidator(respJSON.Data[0], resp.Data[0]); err != nil {
		return errors.Wrapf(err, "incorrect validator data returned from the API request")
	}
	if err = assertValidator(respJSON.Data[1], resp.Data[1]); err != nil {
		return errors.Wrapf(err, "incorrect validator data returned from the API request")
	}
	return nil
}

func assertValidator(jsonVal *validatorContainerJson, val *ethpbv1.ValidatorContainer) error {
	if jsonVal == nil {
		return errors.New("validator is nil")
	}
	if jsonVal.Index != "100" && jsonVal.Index != "200" {
		return fmt.Errorf("unexpected validator index '%s'", jsonVal.Index)
	}
	if jsonVal.Balance != strconv.FormatUint(val.Balance, 10) {
		return buildFieldError("Balance", strconv.FormatUint(val.Balance, 10), jsonVal.Balance)
	}
	if jsonVal.Status != strings.ToLower(val.Status.String()) {
		return buildFieldError("Status", strings.ToLower(val.Status.String()), jsonVal.Status)
	}
	if jsonVal.Validator == nil {
		return errors.New("validator is nil")
	}
	if jsonVal.Validator.PublicKey != hexutil.Encode(val.Validator.Pubkey) {
		return buildFieldError("PublicKey", hexutil.Encode(val.Validator.Pubkey), jsonVal.Validator.PublicKey)
	}
	if jsonVal.Validator.Slashed != val.Validator.Slashed {
		return buildFieldError("Slashed", strconv.FormatBool(val.Validator.Slashed), strconv.FormatBool(jsonVal.Validator.Slashed))
	}
	return nil
}

func withCompareSyncCommittee(beaconNodeIdx int, conn *grpc.ClientConn) error {
	ctx := context.Background()
	beaconClient := service.NewBeaconChainClient(conn)
	genesisData, err := beaconClient.GetGenesis(ctx, &empty.Empty{})
	if err != nil {
		return err
	}
	currentEpoch := slots.EpochsSinceGenesis(genesisData.Data.GenesisTime.AsTime())
	if currentEpoch < params.BeaconConfig().AltairForkEpoch {
		return nil
	}

	type syncCommitteeValidatorsJson struct {
		Validators          []string   `json:"validators"`
		ValidatorAggregates [][]string `json:"validator_aggregates"`
	}
	type syncCommitteesResponseJson struct {
		Data *syncCommitteeValidatorsJson `json:"data"`
	}
	resp, err := beaconClient.ListSyncCommittees(ctx, &ethpbv2.StateSyncCommitteesRequest{
		StateId: []byte("head"),
	})
	if err != nil {
		return err
	}
	respJSON := &syncCommitteesResponseJson{}
	if err := doMiddlewareJSONGetRequest(
		v1MiddlewarePathTemplate,
		"/beacon/states/head/sync_committees",
		beaconNodeIdx,
		respJSON,
	); err != nil {
		return err
	}
	if len(respJSON.Data.Validators) != len(resp.Data.Validators) {
		return fmt.Errorf(
			"API Middleware number of validators %d does not match gRPC %d",
			len(respJSON.Data.Validators),
			len(resp.Data.Validators),
		)
	}
	if len(respJSON.Data.ValidatorAggregates) != len(resp.Data.ValidatorAggregates) {
		return fmt.Errorf(
			"API Middleware number of validator aggregates %d does not match gRPC %d",
			len(respJSON.Data.ValidatorAggregates),
			len(resp.Data.ValidatorAggregates),
		)
	}
	return nil
}

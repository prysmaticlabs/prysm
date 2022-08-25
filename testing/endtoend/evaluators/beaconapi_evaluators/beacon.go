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
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"google.golang.org/grpc"
)

// GET "/eth/v1/beacon/blocks/{block_id}"
// GET "/eth/v1/beacon/blocks/{block_id}/root"
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
	blockroot, err := beaconClient.GetBlockRoot(ctx, &ethpbv1.BlockRequest{
		BlockId: []byte("head"),
	})
	if err != nil {
		return err
	}
	blockrootJSON := &apimiddleware.BlockRootResponseJson{}
	if err := doMiddlewareJSONGetRequest(
		v1MiddlewarePathTemplate,
		"/beacon/blocks/head/root",
		beaconNodeIdx,
		blockrootJSON,
	); err != nil {
		return err
	}
	if hexutil.Encode(blockroot.Data.Root) != blockrootJSON.Data.Root {
		return fmt.Errorf("API Middleware block root  %s does not match gRPC block root %s",
			blockrootJSON.Data.Root,
			hexutil.Encode(blockroot.Data.Root))
	}
	return nil
}

// GET "/eth/v1/beacon/blocks/{block_id}/attestations"
// GET "/eth/v1/beacon/pool/attestations"
func withCompareBlockAttestations(beaconNodeIdx int, conn *grpc.ClientConn) error {
	ctx := context.Background()
	beaconClient := service.NewBeaconChainClient(conn)
	resp, err := beaconClient.ListBlockAttestations(ctx, &ethpbv1.BlockRequest{BlockId: []byte("head")})
	if err != nil {
		return err
	}
	respJSON := &apimiddleware.BlockAttestationsResponseJson{}
	if err := doMiddlewareJSONGetRequest(
		v1MiddlewarePathTemplate,
		"/beacon/blocks/head/attestations",
		beaconNodeIdx,
		respJSON,
	); err != nil {
		return err
	}
	if len(resp.Data) != len(respJSON.Data) {
		return fmt.Errorf("API Middleware attestations length  %d does not match gRPC block signature %d",
			len(respJSON.Data),
			len(resp.Data))
	}
	var slot types.Slot
	var index types.CommitteeIndex
	for i, attest := range resp.Data {
		index, err := strconv.ParseUint(respJSON.Data[i].Data.CommitteeIndex, 10, 64)
		if err != nil {
			return err
		}
		slot, err := strconv.ParseUint(respJSON.Data[i].Data.Slot, 10, 64)
		if err != nil {
			return err
		}
		if uint64(attest.Data.Index) == index &&
			uint64(attest.Data.Slot) == slot &&
			hexutil.Encode(attest.Signature) == respJSON.Data[i].Signature {
			slot = uint64(attest.Data.Slot)
			index = uint64(attest.Data.Index)
		} else {
			return fmt.Errorf("API Middleware attestation response %s does not match gRPC attestation response %s ",
				fmt.Sprintf("index: %d, slot: %d, signature: %s", index, slot, respJSON.Data[i].Signature),
				fmt.Sprintf("index: %d, slot: %d, signature: %s", uint64(attest.Data.Index), uint64(attest.Data.Slot), hexutil.Encode(attest.Signature)))
		}
	}
	poolas, err := beaconClient.ListPoolAttestations(ctx, &ethpbv1.AttestationsPoolRequest{
		Slot:           &slot,
		CommitteeIndex: &index,
	})
	if err != nil {
		return err
	}
	poolJSON := &apimiddleware.AttestationsPoolResponseJson{}
	if err := doMiddlewareJSONGetRequest(
		v1MiddlewarePathTemplate,
		"/beacon/pool/attestations",
		beaconNodeIdx,
		poolJSON,
	); err != nil {
		return err
	}
	if len(poolas.Data) == 0 {
		return nil
	}

	if len(poolas.Data) != len(poolJSON.Data) {
		return fmt.Errorf("API Middleware pool attestation response length %d does not match gRPC pool attestation response length %d ",
			len(poolJSON.Data), len(poolas.Data))
	}
	for i, pattes := range poolas.Data {
		if hexutil.Encode(pattes.Data.BeaconBlockRoot) != poolJSON.Data[i].Data.BeaconBlockRoot {
			return fmt.Errorf("API Middleware pool attestation response BeaconBlockRoot %s does not match gRPC pool attestation response BeaconBlockRoot %s ",
				poolJSON.Data[i].Data.BeaconBlockRoot, hexutil.Encode(pattes.Data.BeaconBlockRoot))
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

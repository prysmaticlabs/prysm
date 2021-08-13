package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/protobuf/ptypes/empty"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"google.golang.org/grpc"
)

type apiComparisonFunc func(beaconNodeIdx int, conn *grpc.ClientConn) error

func main() {
	conn, err := grpc.Dial("localhost:4000", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	if err := apiGatewayV1Alpha1Verify(conn); err != nil {
		panic(err)
	}
}

func apiGatewayV1Alpha1Verify(conns ...*grpc.ClientConn) error {
	for beaconNodeIdx, conn := range conns {
		if err := runAPIComparisonFunctions(
			beaconNodeIdx,
			conn,
			withCompareValidators,
			withCompareChainHead,
		); err != nil {
			return err
		}
	}
	return nil
}

func withCompareValidators(beacon int, conn *grpc.ClientConn) error {
	ctx := context.Background()
	beaconClient := ethpb.NewBeaconChainClient(conn)
	resp, err := beaconClient.ListValidators(ctx, &ethpb.ListValidatorsRequest{
		QueryFilter: &ethpb.ListValidatorsRequest_Genesis{
			Genesis: true,
		},
		PageSize: 4,
	})
	if err != nil {
		return err
	}
	_ = resp
	basePath := fmt.Sprintf("http://localhost:%d/eth/v1alpha1", 3500+beacon)
	httpResp, err := http.Get(
		basePath + "/beacon/validators?genesis=true&page_size=4",
	)
	if err != nil {
		return err
	}
	jsonResp := make(map[string]string)
	if err = json.NewDecoder(httpResp.Body).Decode(&jsonResp); err != nil {
		return err
	}
	if fmt.Sprintf("%d", resp.Epoch) != jsonResp["epoch"] {
		return fmt.Errorf("gRPC got %d, gateway got %s", resp.Epoch, jsonResp["epoch"])
	}
	return nil
}

func withCompareChainHead(idx int, conn *grpc.ClientConn) error {
	type chainHeadResponse struct {
		HeadSlot string
	}
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
	if httpChainHeadResp.HeadSlot != fmt.Sprintf("%d", resp.HeadSlot) {
		return fmt.Errorf("HTTP gateway chainhead %s does not match gRPC chainhead %d", httpChainHeadResp.HeadSlot, resp.HeadSlot)
	}
	return nil
}

func runAPIComparisonFunctions(beaconNodeIdx int, conn *grpc.ClientConn, fs ...apiComparisonFunc) error {
	for _, f := range fs {
		if err := f(beaconNodeIdx, conn); err != nil {
			return err
		}
	}
	return nil
}

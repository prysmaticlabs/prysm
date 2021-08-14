package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/protobuf/ptypes/empty"
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

func withCompareValidators(beaconNodeIdx int, conn *grpc.ClientConn) error {
	type validatorJSON struct {
		PublicKey                  []string `json:"publicKey"`
		WithdrawalCredentials      []string `json:"withdrawalCredentials"`
		EffectiveBalance           string   `json:"effectiveBalance"`
		Slashed                    string   `json:"slashed"`
		ActivationEligibilityEpoch string   `json:"activationEligibilityEpoch"`
		ActivationEpoch            string   `json:"activationEpoch"`
		ExitEpoch                  string   `json:"exitEpoch"`
		WithdrawableEpoch          string   `json:"withdrawableEpoch"`
	}
	type validatorContainerJSON struct {
		Index     string         `json:"index"`
		Validator *validatorJSON `json:"validator"`
	}
	type validatorsResponseJSON struct {
		Epoch         string                    `json:"epoch"`
		ValidatorList []*validatorContainerJSON `json:"validatorList"`
		NextPageToken string                    `json:"nextPageToken"`
		TotalSize     string                    `json:"totalSize"`
	}
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
	validatorsRespJSON := &ethpb.Validators{}
	basePath := fmt.Sprintf("http://localhost:%d/eth/v1alpha1", 3500+beaconNodeIdx)
	httpResp, err := http.Get(
		basePath + "/beacon/validators?genesis=true&page_size=4",
	)
	if err != nil {
		return err
	}
	if err = json.NewDecoder(httpResp.Body).Decode(&validatorsRespJSON); err != nil {
		return err
	}
	_ = resp
	return nil
}

// Compares a regular beacon chain head GET request with no arguments gRPC and gRPC gateway.
func withCompareChainHead(beaconNodeIdx int, conn *grpc.ClientConn) error {
	type chainHeadResponseJSON struct {
		HeadSlot           string `json:"headSlot"`
		HeadEpoch          string `json:"headEpoch"`
		HeadBlockRoot      string `json:"headBlockRoot"`
		FinalizedSlot      string `json:"finalizedSlot"`
		FinalizedEpoch     string `json:"finalizedEpoch"`
		FinalizedBlockRoot string `json:"finalizedBlockRoot"`
		JustifiedSlot      string `json:"justifiedSlot"`
		JustifiedEpoch     string `json:"justifiedEpoch"`
		JustifiedBlockRoot string `json:"justifiedBlockRoot"`
	}
	beaconClient := ethpb.NewBeaconChainClient(conn)
	ctx := context.Background()
	resp, err := beaconClient.GetChainHead(ctx, &empty.Empty{})
	if err != nil {
		return err
	}
	basePath := fmt.Sprintf("http://localhost:%d/eth/v1alpha1", 3500+beaconNodeIdx)
	apiresp, err := http.Get(
		basePath + "/beacon/chainhead",
	)
	if err != nil {
		return err
	}
	httpChainHeadResp := &chainHeadResponseJSON{}
	if err = json.NewDecoder(apiresp.Body).Decode(&httpChainHeadResp); err != nil {
		return err
	}
	if httpChainHeadResp.HeadSlot != fmt.Sprintf("%d", resp.HeadSlot) {
		return fmt.Errorf(
			"HTTP gateway head slot %s does not match gRPC %d",
			httpChainHeadResp.HeadSlot,
			resp.HeadSlot,
		)
	}
	if httpChainHeadResp.HeadEpoch != fmt.Sprintf("%d", resp.HeadEpoch) {
		return fmt.Errorf(
			"HTTP gateway head epoch %s does not match gRPC %d",
			httpChainHeadResp.HeadEpoch,
			resp.HeadEpoch,
		)
	}
	if httpChainHeadResp.HeadBlockRoot != base64.StdEncoding.EncodeToString(resp.HeadBlockRoot) {
		return fmt.Errorf(
			"HTTP gateway head block root %s does not match gRPC %s",
			httpChainHeadResp.HeadBlockRoot,
			resp.HeadBlockRoot,
		)
	}
	if httpChainHeadResp.FinalizedSlot != fmt.Sprintf("%d", resp.FinalizedSlot) {
		return fmt.Errorf(
			"HTTP gateway finalized slot %s does not match gRPC %d",
			httpChainHeadResp.FinalizedSlot,
			resp.FinalizedSlot,
		)
	}
	if httpChainHeadResp.FinalizedEpoch != fmt.Sprintf("%d", resp.FinalizedEpoch) {
		return fmt.Errorf(
			"HTTP gateway finalized epoch %s does not match gRPC %d",
			httpChainHeadResp.FinalizedEpoch,
			resp.FinalizedEpoch,
		)
	}
	if httpChainHeadResp.FinalizedBlockRoot != base64.StdEncoding.EncodeToString(resp.FinalizedBlockRoot) {
		return fmt.Errorf(
			"HTTP gateway finalized block root %s does not match gRPC %s",
			httpChainHeadResp.FinalizedBlockRoot,
			resp.FinalizedBlockRoot,
		)
	}
	if httpChainHeadResp.JustifiedSlot != fmt.Sprintf("%d", resp.JustifiedSlot) {
		return fmt.Errorf(
			"HTTP gateway justified slot %s does not match gRPC %d",
			httpChainHeadResp.FinalizedSlot,
			resp.FinalizedSlot,
		)
	}
	if httpChainHeadResp.JustifiedEpoch != fmt.Sprintf("%d", resp.JustifiedEpoch) {
		return fmt.Errorf(
			"HTTP gateway justified epoch %s does not match gRPC %d",
			httpChainHeadResp.FinalizedEpoch,
			resp.FinalizedEpoch,
		)
	}
	if httpChainHeadResp.JustifiedBlockRoot != base64.StdEncoding.EncodeToString(resp.JustifiedBlockRoot) {
		return fmt.Errorf(
			"HTTP gateway justified block root %s does not match gRPC %s",
			httpChainHeadResp.JustifiedBlockRoot,
			resp.JustifiedBlockRoot,
		)
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

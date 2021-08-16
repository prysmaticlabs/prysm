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

const (
	v1Alpha1GatewayPathTemplate = "http://localhost:%d/eth/v1alpha1"
	v1Alpha1GatewayPort         = 3500
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
			withComparePeers,
			withCompareAttestationPool,
			withCompareValidators,
			withCompareChainHead,
		); err != nil {
			return err
		}
	}
	return nil
}

func withComparePeers(beaconNodeIdx int, conn *grpc.ClientConn) error {
	type peersJSON struct {
		Epoch string `json:"epoch"`
		Root  string `json:"root"`
	}
	type peersResponseJSON struct {
		Peers         []*peersJSON `json:"attestations"`
		NextPageToken string       `json:"nextPageToken"`
		TotalSize     int32        `json:"totalSize"`
	}
	ctx := context.Background()
	nodeClient := ethpb.NewNodeClient(conn)
	resp, err := nodeClient.ListPeers(ctx, &empty.Empty{})
	if err != nil {
		return err
	}
	respJSON := &peersResponseJSON{}
	if err := doGatewayJSONRequest(
		"/node/peers",
		beaconNodeIdx,
		respJSON,
	); err != nil {
		return err
	}
	_ = resp
	return nil
}

func withCompareAttestationPool(beaconNodeIdx int, conn *grpc.ClientConn) error {
	type checkpointJSON struct {
		Epoch string `json:"epoch"`
		Root  string `json:"root"`
	}
	type attestationDataJSON struct {
		Slot            string          `json:"slot"`
		CommitteeIndex  string          `json:"committeeIndex"`
		BeaconBlockRoot string          `json:"beaconBlockRoot"`
		Source          *checkpointJSON `json:"source"`
		Target          *checkpointJSON `json:"target"`
	}
	type attestationJSON struct {
		AggregationBits string               `json:"aggregationBits"`
		Data            *attestationDataJSON `json:"data"`
		Signature       string               `json:"signature"`
	}
	type attestationPoolResponseJSON struct {
		Attestations  []*attestationJSON `json:"attestations"`
		NextPageToken string             `json:"nextPageToken"`
		TotalSize     int32              `json:"totalSize"`
	}
	ctx := context.Background()
	beaconClient := ethpb.NewBeaconChainClient(conn)
	resp, err := beaconClient.AttestationPool(ctx, &ethpb.AttestationPoolRequest{
		PageSize: 4,
	})
	if err != nil {
		return err
	}
	respJSON := &attestationPoolResponseJSON{}
	if err := doGatewayJSONRequest(
		"/beacon/attestations/pool?page_size=4",
		beaconNodeIdx,
		respJSON,
	); err != nil {
		return err
	}

	// Begin comparisons.
	if respJSON.NextPageToken != resp.NextPageToken {
		return fmt.Errorf(
			"HTTP gateway next page token %s does not match gRPC %s",
			respJSON.NextPageToken,
			resp.NextPageToken,
		)
	}
	if respJSON.TotalSize != resp.TotalSize {
		return fmt.Errorf(
			"HTTP gateway total size %d does not match gRPC %d",
			respJSON.TotalSize,
			resp.TotalSize,
		)
	}
	for i, att := range respJSON.Attestations {
		grpcAtt := resp.Attestations[i]
		if att.AggregationBits != fmt.Sprintf("%s", base64.StdEncoding.EncodeToString(grpcAtt.AggregationBits)) {
			return fmt.Errorf(
				"HTTP gateway attestation %d aggregation bits %s does not match gRPC %d",
				i,
				att.AggregationBits,
				grpcAtt.AggregationBits,
			)
		}
		data := att.Data
		grpcData := grpcAtt.Data
		if data.Slot != fmt.Sprintf("%d", grpcData.Slot) {
			return fmt.Errorf(
				"HTTP gateway attestation %d slot %s does not match gRPC %d",
				i,
				data.Slot,
				grpcData.Slot,
			)
		}
		if data.CommitteeIndex != fmt.Sprintf("%d", grpcData.CommitteeIndex) {
			return fmt.Errorf(
				"HTTP gateway attestation %d committee index %s does not match gRPC %d",
				i,
				data.CommitteeIndex,
				grpcData.CommitteeIndex,
			)
		}
		if data.BeaconBlockRoot != fmt.Sprintf("%s", base64.StdEncoding.EncodeToString(grpcData.BeaconBlockRoot)) {
			return fmt.Errorf(
				"HTTP gateway attestation %d beacon block root %s does not match gRPC %d",
				i,
				data.BeaconBlockRoot,
				grpcData.BeaconBlockRoot,
			)
		}
		if data.Source.Epoch != fmt.Sprintf("%d", grpcData.Source.Epoch) {
			return fmt.Errorf(
				"HTTP gateway attestation %d source epoch %s does not match gRPC %d",
				i,
				data.Source.Epoch,
				grpcData.Source.Epoch,
			)
		}
		if data.Source.Root != fmt.Sprintf("%s", base64.StdEncoding.EncodeToString(grpcData.Source.Root)) {
			return fmt.Errorf(
				"HTTP gateway attestation %d source root %s does not match gRPC %d",
				i,
				data.Source.Root,
				grpcData.Source.Root,
			)
		}
		if data.Target.Epoch != fmt.Sprintf("%d", grpcData.Target.Epoch) {
			return fmt.Errorf(
				"HTTP gateway attestation %d target epoch %s does not match gRPC %d",
				i,
				data.Target.Epoch,
				grpcData.Target.Epoch,
			)
		}
		if data.Target.Root != fmt.Sprintf("%s", base64.StdEncoding.EncodeToString(grpcData.Target.Root)) {
			return fmt.Errorf(
				"HTTP gateway attestation %d target root %s does not match gRPC %d",
				i,
				data.Target.Root,
				grpcData.Target.Root,
			)
		}
		if att.Signature != fmt.Sprintf("%s", base64.StdEncoding.EncodeToString(grpcAtt.Signature)) {
			return fmt.Errorf(
				"HTTP gateway attestation %d signature %s does not match gRPC %d",
				i,
				att.Signature,
				grpcAtt.Signature,
			)
		}
	}
	return nil
}

func withCompareValidators(beaconNodeIdx int, conn *grpc.ClientConn) error {
	type validatorJSON struct {
		PublicKey                  string `json:"publicKey"`
		WithdrawalCredentials      string `json:"withdrawalCredentials"`
		EffectiveBalance           string `json:"effectiveBalance"`
		Slashed                    bool   `json:"slashed"`
		ActivationEligibilityEpoch string `json:"activationEligibilityEpoch"`
		ActivationEpoch            string `json:"activationEpoch"`
		ExitEpoch                  string `json:"exitEpoch"`
		WithdrawableEpoch          string `json:"withdrawableEpoch"`
	}
	type validatorContainerJSON struct {
		Index     string         `json:"index"`
		Validator *validatorJSON `json:"validator"`
	}
	type validatorsResponseJSON struct {
		Epoch         string                    `json:"epoch"`
		ValidatorList []*validatorContainerJSON `json:"validatorList"`
		NextPageToken string                    `json:"nextPageToken"`
		TotalSize     int32                     `json:"totalSize"`
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
	respJSON := &validatorsResponseJSON{}
	if err := doGatewayJSONRequest(
		"/validators?genesis=true&page_size=4",
		beaconNodeIdx,
		respJSON,
	); err != nil {
		return err
	}

	// Begin comparisons.
	if respJSON.Epoch != fmt.Sprintf("%d", resp.Epoch) {
		return fmt.Errorf(
			"HTTP gateway epoch %s does not match gRPC %d",
			respJSON.Epoch,
			resp.Epoch,
		)
	}
	if respJSON.NextPageToken != resp.NextPageToken {
		return fmt.Errorf(
			"HTTP gateway next page token %s does not match gRPC %s",
			respJSON.NextPageToken,
			resp.NextPageToken,
		)
	}
	if respJSON.TotalSize != resp.TotalSize {
		return fmt.Errorf(
			"HTTP gateway total size %d does not match gRPC %d",
			respJSON.TotalSize,
			resp.TotalSize,
		)
	}

	// Compare validators.
	for i, val := range respJSON.ValidatorList {
		if val.Index != fmt.Sprintf("%d", resp.ValidatorList[i].Index) {
			return fmt.Errorf(
				"HTTP gateway validator %d index %s does not match gRPC %d",
				i,
				val.Index,
				resp.ValidatorList[i].Index,
			)
		}
		httpVal := val.Validator
		grpcVal := resp.ValidatorList[i].Validator
		if httpVal.PublicKey != fmt.Sprintf("%s", base64.StdEncoding.EncodeToString(grpcVal.PublicKey)) {
			return fmt.Errorf(
				"HTTP gateway validator %d public key %s does not match gRPC %d",
				i,
				httpVal.PublicKey,
				grpcVal.PublicKey,
			)
		}
		continue
	}
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
	respJSON := &chainHeadResponseJSON{}
	if err := doGatewayJSONRequest(
		"/beacon/chainhead",
		beaconNodeIdx,
		respJSON,
	); err != nil {
		return err
	}

	if respJSON.HeadSlot != fmt.Sprintf("%d", resp.HeadSlot) {
		return fmt.Errorf(
			"HTTP gateway head slot %s does not match gRPC %d",
			respJSON.HeadSlot,
			resp.HeadSlot,
		)
	}
	if respJSON.HeadEpoch != fmt.Sprintf("%d", resp.HeadEpoch) {
		return fmt.Errorf(
			"HTTP gateway head epoch %s does not match gRPC %d",
			respJSON.HeadEpoch,
			resp.HeadEpoch,
		)
	}
	if respJSON.HeadBlockRoot != base64.StdEncoding.EncodeToString(resp.HeadBlockRoot) {
		return fmt.Errorf(
			"HTTP gateway head block root %s does not match gRPC %s",
			respJSON.HeadBlockRoot,
			resp.HeadBlockRoot,
		)
	}
	if respJSON.FinalizedSlot != fmt.Sprintf("%d", resp.FinalizedSlot) {
		return fmt.Errorf(
			"HTTP gateway finalized slot %s does not match gRPC %d",
			respJSON.FinalizedSlot,
			resp.FinalizedSlot,
		)
	}
	if respJSON.FinalizedEpoch != fmt.Sprintf("%d", resp.FinalizedEpoch) {
		return fmt.Errorf(
			"HTTP gateway finalized epoch %s does not match gRPC %d",
			respJSON.FinalizedEpoch,
			resp.FinalizedEpoch,
		)
	}
	if respJSON.FinalizedBlockRoot != base64.StdEncoding.EncodeToString(resp.FinalizedBlockRoot) {
		return fmt.Errorf(
			"HTTP gateway finalized block root %s does not match gRPC %s",
			respJSON.FinalizedBlockRoot,
			resp.FinalizedBlockRoot,
		)
	}
	if respJSON.JustifiedSlot != fmt.Sprintf("%d", resp.JustifiedSlot) {
		return fmt.Errorf(
			"HTTP gateway justified slot %s does not match gRPC %d",
			respJSON.FinalizedSlot,
			resp.FinalizedSlot,
		)
	}
	if respJSON.JustifiedEpoch != fmt.Sprintf("%d", resp.JustifiedEpoch) {
		return fmt.Errorf(
			"HTTP gateway justified epoch %s does not match gRPC %d",
			respJSON.FinalizedEpoch,
			resp.FinalizedEpoch,
		)
	}
	if respJSON.JustifiedBlockRoot != base64.StdEncoding.EncodeToString(resp.JustifiedBlockRoot) {
		return fmt.Errorf(
			"HTTP gateway justified block root %s does not match gRPC %s",
			respJSON.JustifiedBlockRoot,
			resp.JustifiedBlockRoot,
		)
	}
	return nil
}

func doGatewayJSONRequest(requestPath string, beaconNodeIdx int, dst interface{}) error {
	basePath := fmt.Sprintf(v1Alpha1GatewayPathTemplate, v1Alpha1GatewayPort+beaconNodeIdx)
	httpResp, err := http.Get(
		basePath + requestPath,
	)
	if err != nil {
		return err
	}
	if err = json.NewDecoder(httpResp.Body).Decode(&dst); err != nil {
		return err
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

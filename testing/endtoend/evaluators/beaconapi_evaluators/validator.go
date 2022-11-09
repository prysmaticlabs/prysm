package beaconapi_evaluators

import (
	"google.golang.org/grpc"
)

func withCompareAttesterDuties(beaconNodeIdx int, conn *grpc.ClientConn) error {
	//ctx := context.Background()
	//beaconClient := service.NewBeaconChainClient(conn)
	//genesisData, err := beaconClient.GetGenesis(ctx, &empty.Empty{})
	//if err != nil {
	//	return err
	//}
	//currentEpoch := slots.EpochsSinceGenesis(genesisData.Data.GenesisTime.AsTime())
	//if currentEpoch < params.BeaconConfig().AltairForkEpoch {
	//	return nil
	//}
	//validatorClient := service.NewBeaconValidatorClient(conn)
	//resp, err := validatorClient.GetAttesterDuties(ctx, &ethpbv1.AttesterDutiesRequest{
	//	Epoch: helpers.AltairE2EForkEpoch,
	//	Index: []types.ValidatorIndex{0},
	//})
	//if err != nil {
	//	return err
	//}
	//// We post a top-level array, not an object, as per the spec.
	//reqJSON := []string{"0"}
	//respJSON := &appimiddleware.attesterDutiesResponseJson{}
	//if err := doMiddlewareJSONPostRequestV1(
	//	"/validator/duties/attester/"+strconv.Itoa(helpers.AltairE2EForkEpoch),
	//	beaconNodeIdx,
	//	reqJSON,
	//	respJSON,
	//); err != nil {
	//	return err
	//}
	//if respJSON.DependentRoot != hexutil.Encode(resp.DependentRoot) {
	//	return buildFieldError("DependentRoot", string(resp.DependentRoot), respJSON.DependentRoot)
	//}
	//if len(respJSON.Data) != len(resp.Data) {
	//	return fmt.Errorf(
	//		"API Middleware number of duties %d does not match gRPC %d",
	//		len(respJSON.Data),
	//		len(resp.Data),
	//	)
	//}
	return nil
}

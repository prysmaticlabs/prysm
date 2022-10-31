package beaconapi_evaluators

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"google.golang.org/grpc"
)

// https://ethereum.github.io/beacon-APIs/#/Debug/getStateV2
func withCompareDebugState(beaconNodeIdx int, conn *grpc.ClientConn) error {
	ctx := context.Background()
	beaconClient := service.NewBeaconChainClient(conn)
	debugClient := service.NewBeaconDebugClient(conn)
	genesisData, err := beaconClient.GetGenesis(ctx, &empty.Empty{})
	if err != nil {
		return errors.Wrap(err, "error getting genesis data")
	}
	currentEpoch := slots.EpochsSinceGenesis(genesisData.Data.GenesisTime.AsTime())
	respJSONPrysm := &ethpbv2.BeaconStateResponseV2{}
	respJSONLighthouse := &ethpbv2.BeaconStateResponseV2{}
	var check string
	if currentEpoch < 4 {
		check = "genesis"
	} else {
		check = "finalized"
	}
	grpcResp, err := debugClient.GetBeaconStateV2(ctx, &ethpbv2.BeaconStateRequestV2{StateId: []byte(check)})
	if err != nil {
		return errors.Wrap(err, "BeaconStateV2 errors")
	}
	if grpcResp == nil {
		return errors.New("BeaconStateV2 empty")
	}
	if err := doMiddlewareJSONGetRequest(
		v2MiddlewarePathTemplate,
		"/debug/beacon/states/head",
		beaconNodeIdx,
		respJSONPrysm,
	); err != nil {
		return errors.Wrap(err, "prysm json error")
	}
	if err := doMiddlewareJSONGetRequest(
		v2MiddlewarePathTemplate,
		"/debug/beacon/states/head",
		beaconNodeIdx,
		respJSONLighthouse,
		"lighthouse",
	); err != nil {
		return errors.Wrap(err, "lighthouse json error")
	}
	if !reflect.DeepEqual(respJSONPrysm, respJSONLighthouse) {
		p, err := json.Marshal(respJSONPrysm)
		if err != nil {
			return errors.Wrap(err, "prysm json")
		}
		l, err := json.Marshal(respJSONLighthouse)
		if err != nil {
			return errors.Wrap(err, "lighthouse json")
		}
		return fmt.Errorf("prysm response %s does not match lighthouse response %s",
			string(p),
			string(l))
	}
	return nil
}

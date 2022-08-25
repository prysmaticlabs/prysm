package beaconapi_evaluators

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	"google.golang.org/grpc"
)

func withCompareNodeMetaData(beaconNodeIdx int, conn *grpc.ClientConn) error {
	ctx := context.Background()
	nodeClient := service.NewBeaconNodeClient(conn)
	if err := compareNodeIdentity(ctx, beaconNodeIdx, nodeClient); err != nil {
		return err
	}
	if err := compareNodePeers(ctx, beaconNodeIdx, nodeClient); err != nil {
		return err
	}
	return nil
}

// /eth/v1/node/identity
func compareNodeIdentity(ctx context.Context, beaconNodeIdx int, nodeClient service.BeaconNodeClient) error {
	resp, err := nodeClient.GetIdentity(ctx, &empty.Empty{})
	if err != nil {
		return err
	}

	respJSON := &apimiddleware.IdentityResponseJson{}
	if err := doMiddlewareJSONGetRequest(
		v1MiddlewarePathTemplate,
		"/node/identity",
		beaconNodeIdx,
		respJSON,
	); err != nil {
		return err
	}
	if resp.Data.Enr != respJSON.Data.Enr {
		return fmt.Errorf("API Middleware node identity  %s does not match gRPC node identity %s",
			respJSON.Data.Enr,
			resp.Data.Enr)
	}
	return nil
}

// /eth/v1/node/peers
func compareNodePeers(ctx context.Context, beaconNodeIdx int, nodeClient service.BeaconNodeClient) error {
	resp, err := nodeClient.ListPeers(ctx, &ethpbv1.PeersRequest{
		State:     []ethpbv1.ConnectionState{ethpbv1.ConnectionState_CONNECTING},
		Direction: []ethpbv1.PeerDirection{ethpbv1.PeerDirection_INBOUND},
	})
	if err != nil {
		return err
	}

	respJSON := &apimiddleware.PeersResponseJson{}
	if err := doMiddlewareJSONGetRequest(
		v1MiddlewarePathTemplate,
		"/node/peers",
		beaconNodeIdx,
		respJSON,
	); err != nil {
		return err
	}

	for i, peer := range resp.Data {
		if peer.PeerId != respJSON.Data[i].PeerId {
			return fmt.Errorf("API Middleware peer id  %s does not match gRPC peer id %s",
				respJSON.Data[i].PeerId,
				peer.PeerId)
		}
	}

	return nil
}

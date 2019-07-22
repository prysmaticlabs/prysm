package rpc

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// NodeServer defines a server implementation of the gRPC Node service,
// providing RPC endpoints for verifying a beacon node's sync status, genesis and
// version information, and services the node implements and runs.
type NodeServer struct{}

// GetSyncStatus checks the current network sync status of the node.
func (ns *NodeServer) GetSyncStatus(ctx context.Context, _ *ptypes.Empty) (*ethpb.SyncStatus, error) {
	return nil, nil
}

// GetVersion checks the version information of the beacon node.
func (ns *NodeServer) GetVersion(ctx context.Context, _ *ptypes.Empty) (*ethpb.Version, error) {
	return nil, nil
}

// ListImplementedServices lists the services implemented and enabled by this node.
//
// Any service not present in this list may return UNIMPLEMENTED or
// PERMISSION_DENIED. The server may also support fetching services by grpc
// reflection.
func (ns *NodeServer) ListImplementedServices(ctx context.Context, _ *ptypes.Empty) (*ethpb.ImplementedServices, error) {
	return nil, nil
}

package beacon

import (
	"context"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetBeaconState retrieves a beacon state
// from the beacon node by either a slot or block root.
func (bs *Server) GetBeaconState(
	ctx context.Context,
	req *pbrpc.BeaconStateRequest,
) (*pbp2p.BeaconState, error) {
	if !featureconfig.Get().NewStateMgmt {
		return nil, status.Error(codes.FailedPrecondition, "requires --enable-new-state-mgmt to function")
	}
	switch q := req.QueryFilter.(type) {
	case *pbrpc.BeaconStateRequest_Slot:
		st, err := bs.StateGen.StateBySlot(ctx, q.Slot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not compute state by slot: %v", err)
		}
		return st.CloneInnerState(), nil
	case *pbrpc.BeaconStateRequest_BlockRoot:
		st, err := bs.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(q.BlockRoot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not compute state by block root: %v", err)
		}
		return st.CloneInnerState(), nil
	default:
		return nil, status.Error(codes.InvalidArgument, "need to specify either a block root or slot to request state")
	}
}

package beacon

import (
	"context"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	//"google.golang.org/grpc/codes"
	//"google.golang.org/grpc/status"
)

// GetBeaconState --
func (bs *Server) GetBeaconState(
	ctx context.Context,
	req *pbrpc.BeaconStateRequest,
) (*pbp2p.BeaconState, error) {

	//currentSlot := bs.GenesisTimeFetcher.CurrentSlot()
	//var requestedSlot uint64
	//switch q := req.QueryFilter.(type) {
	//case *ethpb.ListCommitteesRequest_Epoch:
	//	requestedSlot = helpers.StartSlot(q.Epoch)
	//case *ethpb.ListCommitteesRequest_Genesis:
	//	requestedSlot = 0
	//default:
	//	requestedSlot = currentSlot
	//}
	return nil, nil
}

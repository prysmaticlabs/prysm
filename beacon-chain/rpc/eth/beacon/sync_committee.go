package beacon

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/proto/eth/v2"
)

func (bs *Server) ListSyncCommittees(ctx context.Context, request *eth.StateSyncCommitteesRequest) (*eth.StateSyncCommitteesResponse, error) {
	panic("implement me")
}

func (bs *Server) SubmitSyncCommitteeSignature(ctx context.Context, message *eth.SyncCommitteeMessage) (*empty.Empty, error) {
	panic("implement me")
}

package beacon

import (
	"context"

	"github.com/pkg/errors"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
)

func (bs *Server) GetBlobsSidecar(ctx context.Context, req *ethpbv1.BlobsRequest) (*ethpbv1.BlobsResponse, error) {
	// TODO: implement this with new blob type request
	return nil, errors.New("not implemented")
}

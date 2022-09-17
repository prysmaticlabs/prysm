package beacon

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
)

func (bs *Server) GetBlobsSidecar(ctx context.Context, req *ethpbv1.BlobsRequest) (*ethpbv1.BlobsResponse, error) {
	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, errors.Wrap(err, "GetBlobs")
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to htr block")
	}
	sidecar, err := bs.BeaconDB.BlobsSidecar(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("failed to get blobs sidecar for block %x", root)
	}
	var blobs []*ethpbv1.Blob
	for _, b := range sidecar.Blobs {
		var data []byte
		// go through each element, concat them
		for _, el := range b.Blob {
			data = append(data, el...)
		}
		blobs = append(blobs, &ethpbv1.Blob{Data: data})
	}
	return &ethpbv1.BlobsResponse{
		BeaconBlockRoot: sidecar.BeaconBlockRoot,
		BeaconBlockSlot: uint64(sidecar.BeaconBlockSlot),
		Blobs:           blobs,
		AggregatedProof: sidecar.AggregatedProof,
	}, nil
}

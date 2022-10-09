package beacon

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
)

func (bs *Server) GetBlobsSidecar(ctx context.Context, req *ethpbv1.BlobsRequest) (*ethpbv1.BlobsResponse, error) {
	sblk, err := bs.blockFromBlockID(ctx, req.BlockId)
	err = handleGetBlockError(sblk, err)
	if err != nil {
		return nil, errors.Wrap(err, "GetBlobs")
	}
	block := sblk.Block()
	root, err := block.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to htr block")
	}
	sidecar, err := bs.BeaconDB.BlobsSidecar(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("failed to get blobs sidecar for block %x", root)
	}
	var blobs []*ethpbv1.Blob
	var aggregatedProof []byte
	if sidecar != nil {
		aggregatedProof = sidecar.AggregatedProof
		for _, b := range sidecar.Blobs {
			var data []byte
			// go through each element, concat them
			for _, el := range b.Blob {
				data = append(data, el...)
			}
			blobs = append(blobs, &ethpbv1.Blob{Data: data})
		}
	}
	return &ethpbv1.BlobsResponse{
		BeaconBlockRoot: root[:],
		BeaconBlockSlot: uint64(block.Slot()),
		Blobs:           blobs,
		AggregatedProof: aggregatedProof,
	}, nil
}

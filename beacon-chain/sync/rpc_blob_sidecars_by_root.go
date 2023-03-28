package sync

import (
	"context"

	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/monitoring/tracing"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
)

func blobMinReqEpoch(finalized, current primitives.Epoch) primitives.Epoch {
	// max(finalized_epoch, current_epoch - MIN_EPOCHS_FOR_BLOB_SIDECARS_REQUESTS, DENEB_FORK_EPOCH)
	denebFork := params.BeaconConfig().DenebForkEpoch
	var reqWindow primitives.Epoch
	if current > params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest {
		reqWindow = current - params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest
	}
	if finalized >= reqWindow && finalized > denebFork {
		return finalized
	}
	if reqWindow >= finalized && reqWindow > denebFork {
		return reqWindow
	}
	return denebFork
}

// blobSidecarByRootRPCHandler handles the /eth2/beacon_chain/req/blob_sidecars_by_root/1/ RPC request.
// spec: https://github.com/ethereum/consensus-specs/blob/a7e45db9ac2b60a33e144444969ad3ac0aae3d4c/specs/deneb/p2p-interface.md#blobsidecarsbyroot-v1
func (s *Service) blobSidecarByRootRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.blobSidecarByRootRPCHandler")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, ttfbTimeout)
	defer cancel()
	SetRPCStreamDeadlines(stream)
	log := log.WithField("handler", p2p.BlobSidecarsByRootName[1:]) // slice the leading slash off the name var
	ref, ok := msg.(*types.BlobSidecarsByRootReq)
	if !ok {
		return errors.New("message is not type BlobSidecarsByRootReq")
	}
	minReqEpoch := blobMinReqEpoch(s.cfg.chain.FinalizedCheckpt().Epoch, slots.ToEpoch(s.cfg.chain.CurrentSlot()))
	blobIdents := *ref
	for i := range blobIdents {
		root, idx := bytesutil.ToBytes32(blobIdents[i].BlockRoot), blobIdents[i].Index
		scs, err := s.cfg.beaconDB.BlobSidecarsByRoot(ctx, root)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				log.WithError(err).Errorf("error retrieving BlobSidecar, root=%x, index=%d", root, idx)
				continue
			}
			log.WithError(err).Debugf("error retrieving BlobSidecar, root=%x, index=%d", root, idx)
			s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream)
			return err
		}
		if idx >= uint64(len(scs)) {
			continue
		}
		sc := scs[idx]

		// If any root in the request content references a block earlier than minimum_request_epoch,
		// peers MAY respond with error code 3: ResourceUnavailable or not include the blob in the response.
		if slots.ToEpoch(sc.Slot) < minReqEpoch {
			s.writeErrorResponseToStream(responseCodeResourceUnavailable, types.ErrBlobLTMinRequest.Error(), stream)
			log.WithError(types.ErrBlobLTMinRequest).
				Debugf("requested blob for block %#x before minimum_request_epoch", blobIdents[i].BlockRoot)
			return types.ErrBlobLTMinRequest
		}
		SetStreamWriteDeadline(stream, defaultWriteDuration)
		if chunkErr := WriteBlobSidecarChunk(stream, s.cfg.chain, s.cfg.p2p.Encoding(), sc); chunkErr != nil {
			log.WithError(chunkErr).Debug("Could not send a chunked response")
			s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream)
			tracing.AnnotateError(span, chunkErr)
			return chunkErr
		}
		s.rateLimiter.add(stream, 1)
	}
	closeStream(stream, log)
	return nil
}

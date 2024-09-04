package sync

import (
	"context"
	"fmt"

	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/pkg/errors"
	light_client "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/light-client"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	"github.com/sirupsen/logrus"
)

// blobSidecarByRootRPCHandler handles the /eth2/beacon_chain/req/blob_sidecars_by_root/1/ RPC request.
// spec: https://github.com/ethereum/consensus-specs/blob/a7e45db9ac2b60a33e144444969ad3ac0aae3d4c/specs/deneb/p2p-interface.md#blobsidecarsbyroot-v1
func (s *Service) lightClientBootstrapRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.lightClientBootstrapRPCHandler")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, ttfbTimeout)
	defer cancel()
	SetRPCStreamDeadlines(stream)
	log := log.WithField("handler", p2p.LightClientBootstrapName[1:]) // slice the leading slash off the name var
	req, ok := msg.(*types.LightClientBootstrapReq)
	if !ok {
		return errors.New("message is not type LightClientBootstrapReq")
	}

	// TODO: rate limit?
	block, err := s.cfg.beaconDB.Block(ctx, *req)
	if err != nil {
		s.writeErrorResponseToStream(responseCodeServerError)
	} else if block.IsNil() {
		s.writeErrorResponseToStream(responseCodeInvalidRequest)
	}

	light_client.CreateLightClientBootstrap(ctx)

	for i := range blobIdents {
		if err := ctx.Err(); err != nil {
			closeStream(stream, log)
			return err
		}

		// Throttle request processing to no more than batchSize/sec.
		if i != 0 && i%batchSize == 0 && ticker != nil {
			<-ticker.C
		}
		s.rateLimiter.add(stream, 1)
		root, idx := bytesutil.ToBytes32(blobIdents[i].BlockRoot), blobIdents[i].Index
		sc, err := s.cfg.blobStorage.Get(root, idx)
		if err != nil {
			if db.IsNotFound(err) {
				log.WithError(err).WithFields(logrus.Fields{
					"root":  fmt.Sprintf("%#x", root),
					"index": idx,
				}).Debugf("Peer requested blob sidecar by root not found in db")
				continue
			}
			log.WithError(err).Errorf("unexpected db error retrieving BlobSidecar, root=%x, index=%d", root, idx)
			s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream)
			return err
		}

		// If any root in the request content references a block earlier than minimum_request_epoch,
		// peers MAY respond with error code 3: ResourceUnavailable or not include the blob in the response.
		// note: we are deviating from the spec to allow requests for blobs that are before minimum_request_epoch,
		// up to the beginning of the retention period.
		if sc.Slot() < minReqSlot {
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
	}
	closeStream(stream, log)
	return nil
}

func validateBlobByRootRequest(blobIdents types.BlobSidecarsByRootReq) error {
	if uint64(len(blobIdents)) > params.BeaconConfig().MaxRequestBlobSidecars {
		return types.ErrMaxBlobReqExceeded
	}
	return nil
}

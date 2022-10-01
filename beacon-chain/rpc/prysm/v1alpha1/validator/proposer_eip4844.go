package validator

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition/interop"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/sirupsen/logrus"
)

func (vs *Server) getEip4844BeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.GenericBeaconBlock, error) {
	// This routine is similar to buildBellatrixBeaconBlock. But it also needs to be backwards compatible with bellatrix payloads when necessary

	altairBlk, err := vs.BuildAltairBeaconBlock(ctx, req)
	if err != nil {
		return nil, err
	}

	registered, err := vs.validatorRegistered(ctx, altairBlk.ProposerIndex)
	if registered && err == nil {
		// TODO(EIP-4844): Blinded EIP4844 blocks doesn't quite work yet
		/*
			builderReady, b, err := vs.GetAndBuildBlindBlock(ctx, altairBlk)
			if err != nil {
				// In the event of an error, the node should fall back to default execution engine for building block.
				log.WithError(err).Error("Failed to build a block from external builder, falling " +
					"back to local execution client")
				builderGetPayloadMissCount.Inc()
			} else if builderReady {
				return b, nil
			}
		*/
	} else if err != nil {
		log.WithFields(logrus.Fields{
			"slot":           req.Slot,
			"validatorIndex": altairBlk.ProposerIndex,
		}).Errorf("Could not determine validator has registered. Default to local execution client: %v", err)
	}

	execData, payloadID, err := vs.getExecutionPayload(ctx, req.Slot, altairBlk.ProposerIndex, bytesutil.ToBytes32(altairBlk.ParentRoot))
	if err != nil {
		return nil, err
	}
	blobsBundle, err := vs.ExecutionEngineCaller.GetBlobsBundle(ctx, payloadID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get blobs")
	}
	// sanity check the blobs bundle
	if !bytes.Equal(blobsBundle.BlockHash, execData.BlockHash()) {
		return nil, errors.New("invalid blobs bundle received")
	}
	if len(blobsBundle.Blobs) != len(blobsBundle.Kzgs) {
		return nil, errors.New("mismatched blobs and kzgs length")
	}

	switch execData.Version() {
	case version.Bellatrix:
		payload, err := execData.PbGenericPayload()
		if err != nil {
			return nil, err
		}
		return vs.assembleEip4844CompatBlock(ctx, altairBlk, payload, blobsBundle)
	case version.EIP4844:
		payload, err := execData.PbEip4844Payload()
		if err != nil {
			return nil, err
		}
		return vs.assembleEip4844Block(ctx, altairBlk, payload, blobsBundle)
	default:
		return nil, errors.New("unknown payload version received from engine")
	}
}

func (vs *Server) assembleEip4844Block(ctx context.Context, altairBlk *ethpb.BeaconBlockAltair, payload *enginev1.ExecutionPayload4844, blobsBundle *enginev1.BlobsBundle) (*ethpb.GenericBeaconBlock, error) {
	blk := &ethpb.BeaconBlockWithBlobKZGs{
		Slot:          altairBlk.Slot,
		ProposerIndex: altairBlk.ProposerIndex,
		ParentRoot:    altairBlk.ParentRoot,
		StateRoot:     params.BeaconConfig().ZeroHash[:],
		Body: &ethpb.BeaconBlockBodyWithBlobKZGs{
			RandaoReveal:      altairBlk.Body.RandaoReveal,
			Eth1Data:          altairBlk.Body.Eth1Data,
			Graffiti:          altairBlk.Body.Graffiti,
			ProposerSlashings: altairBlk.Body.ProposerSlashings,
			AttesterSlashings: altairBlk.Body.AttesterSlashings,
			Attestations:      altairBlk.Body.Attestations,
			Deposits:          altairBlk.Body.Deposits,
			VoluntaryExits:    altairBlk.Body.VoluntaryExits,
			SyncAggregate:     altairBlk.Body.SyncAggregate,
			ExecutionPayload:  payload,
			BlobKzgs:          blobsBundle.Kzgs,
		},
	}

	// Compute state root with the newly constructed block.
	signedBlk := &ethpb.SignedBeaconBlockWithBlobKZGs{Block: blk, Signature: make([]byte, 96)}
	wsb, err := consensusblocks.NewSignedBeaconBlock(signedBlk)
	if err != nil {
		return nil, err
	}
	stateRoot, err := vs.computeStateRoot(ctx, wsb)
	if err != nil {
		interop.WriteBlockToDisk(wsb, true /*failed*/)
		return nil, fmt.Errorf("could not compute state root: %v", err)
	}
	blk.StateRoot = stateRoot

	r, err := blk.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	// TOOD(EIP-4844): Upgrade Hazard. If only the CL supports new payloads, how should the beacon block root be computed? Right now the spec says
	// it should use the new payload regardless. But then validators will have to be careful and detect this to use the post-4844 block when voting/attesting.

	var sideCar *ethpb.BlobsSidecar
	if len(blobsBundle.Blobs) != 0 {
		sideCar = &ethpb.BlobsSidecar{
			BeaconBlockRoot: r[:],
			BeaconBlockSlot: wsb.Block().Slot(),
			Blobs:           blobsBundle.Blobs,
			AggregatedProof: blobsBundle.AggregatedProof,
		}
	}

	return &ethpb.GenericBeaconBlock{
		Block:   &ethpb.GenericBeaconBlock_Eip4844{Eip4844: blk},
		Sidecar: sideCar,
	}, nil
}

func (vs *Server) assembleEip4844CompatBlock(ctx context.Context, altairBlk *ethpb.BeaconBlockAltair, payload *enginev1.ExecutionPayload, blobsBundle *enginev1.BlobsBundle) (*ethpb.GenericBeaconBlock, error) {
	blk := &ethpb.BeaconBlockWithBlobKZGsCompat{
		Slot:          altairBlk.Slot,
		ProposerIndex: altairBlk.ProposerIndex,
		ParentRoot:    altairBlk.ParentRoot,
		StateRoot:     params.BeaconConfig().ZeroHash[:],
		Body: &ethpb.BeaconBlockBodyWithBlobKZGsCompat{
			RandaoReveal:      altairBlk.Body.RandaoReveal,
			Eth1Data:          altairBlk.Body.Eth1Data,
			Graffiti:          altairBlk.Body.Graffiti,
			ProposerSlashings: altairBlk.Body.ProposerSlashings,
			AttesterSlashings: altairBlk.Body.AttesterSlashings,
			Attestations:      altairBlk.Body.Attestations,
			Deposits:          altairBlk.Body.Deposits,
			VoluntaryExits:    altairBlk.Body.VoluntaryExits,
			SyncAggregate:     altairBlk.Body.SyncAggregate,
			ExecutionPayload:  payload,
			BlobKzgs:          blobsBundle.Kzgs,
		},
	}

	// Compute state root with the newly constructed block.
	signedBlk := &ethpb.SignedBeaconBlockWithBlobKZGsCompat{Block: blk, Signature: make([]byte, 96)}
	wsb, err := consensusblocks.NewSignedBeaconBlock(signedBlk)
	if err != nil {
		return nil, err
	}
	stateRoot, err := vs.computeStateRoot(ctx, wsb)
	if err != nil {
		interop.WriteBlockToDisk(wsb, true /*failed*/)
		return nil, fmt.Errorf("could not compute state root: %v", err)
	}
	blk.StateRoot = stateRoot

	r, err := wsb.Block().HashTreeRoot()
	if err != nil {
		return nil, err
	}

	var sideCar *ethpb.BlobsSidecar
	if len(blobsBundle.Blobs) != 0 {
		sideCar = &ethpb.BlobsSidecar{
			BeaconBlockRoot: r[:],
			BeaconBlockSlot: wsb.Block().Slot(),
			Blobs:           blobsBundle.Blobs,
			AggregatedProof: blobsBundle.AggregatedProof,
		}
	}

	return &ethpb.GenericBeaconBlock{
		Block:   &ethpb.GenericBeaconBlock_Eip4844Compat{Eip4844Compat: blk},
		Sidecar: sideCar,
	}, nil
}

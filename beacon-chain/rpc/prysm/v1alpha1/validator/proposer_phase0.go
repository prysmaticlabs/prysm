package validator

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/protolambda/go-kzg/eth"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition/interop"
	v "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blobs"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"go.opencensus.io/trace"
)

// blockData required to create a beacon block.
type blockData struct {
	Slot                  types.Slot
	ParentRoot            []byte
	Graffiti              [32]byte
	ProposerIdx           types.ValidatorIndex
	Eth1Data              *ethpb.Eth1Data
	Deposits              []*ethpb.Deposit
	Attestations          []*ethpb.Attestation
	RandaoReveal          []byte
	ProposerSlashings     []*ethpb.ProposerSlashing
	AttesterSlashings     []*ethpb.AttesterSlashing
	VoluntaryExits        []*ethpb.SignedVoluntaryExit
	SyncAggregate         *ethpb.SyncAggregate
	ExecutionPayload      *enginev1.ExecutionPayload
	ExecutionPayloadV2    *enginev1.ExecutionPayloadCapella
	BlsToExecutionChanges []*ethpb.SignedBLSToExecutionChange
	BlobsKzg              [][]byte
}

func (vs *Server) getPhase0BeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.getPhase0BeaconBlock")
	defer span.End()
	blkData, err := vs.buildPhase0BlockData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("could not build block data: %v", err)
	}

	// Use zero hash as stub for state root to compute later.
	stateRoot := params.BeaconConfig().ZeroHash[:]

	blk := &ethpb.BeaconBlock{
		Slot:          blkData.Slot,
		ParentRoot:    blkData.ParentRoot,
		StateRoot:     stateRoot,
		ProposerIndex: blkData.ProposerIdx,
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:          blkData.Eth1Data,
			Deposits:          blkData.Deposits,
			Attestations:      blkData.Attestations,
			RandaoReveal:      blkData.RandaoReveal,
			ProposerSlashings: blkData.ProposerSlashings,
			AttesterSlashings: blkData.AttesterSlashings,
			VoluntaryExits:    blkData.VoluntaryExits,
			Graffiti:          blkData.Graffiti[:],
		},
	}

	// Compute state root with the newly constructed block.
	wsb, err := consensusblocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: blk, Signature: make([]byte, 96)})
	if err != nil {
		return nil, err
	}
	stateRoot, err = vs.computeStateRoot(ctx, wsb)
	if err != nil {
		interop.WriteBlockToDisk(wsb, true /*failed*/)
		return nil, errors.Wrap(err, "could not compute state root")
	}
	blk.StateRoot = stateRoot
	return blk, nil
}

// Build data required for creating a new beacon block, so this method can be shared across forks.
func (vs *Server) buildPhase0BlockData(ctx context.Context, req *ethpb.BlockRequest) (*blockData, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.buildPhase0BlockData")
	defer span.End()

	if vs.SyncChecker.Syncing() {
		return nil, fmt.Errorf("syncing to latest head, not ready to respond")
	}

	if err := vs.HeadUpdater.UpdateHead(ctx); err != nil {
		log.WithError(err).Error("Could not process attestations and update head")
	}

	// Retrieve the parent block as the current head of the canonical chain.
	parentRoot, err := vs.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve head root: %v", err)
	}

	head, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get head state %v", err)
	}

	head, err = transition.ProcessSlotsUsingNextSlotCache(ctx, head, parentRoot, req.Slot)
	if err != nil {
		return nil, fmt.Errorf("could not advance slots to calculate proposer index: %v", err)
	}

	eth1Data, err := vs.eth1DataMajorityVote(ctx, head)
	if err != nil {
		return nil, fmt.Errorf("could not get ETH1 data: %v", err)
	}

	deposits, atts, err := vs.packDepositsAndAttestations(ctx, head, eth1Data)
	if err != nil {
		return nil, err
	}

	graffiti := bytesutil.ToBytes32(req.Graffiti)

	// Calculate new proposer index.
	idx, err := helpers.BeaconProposerIndex(ctx, head)
	if err != nil {
		return nil, fmt.Errorf("could not calculate proposer index %v", err)
	}

	proposerSlashings := vs.SlashingsPool.PendingProposerSlashings(ctx, head, false /*noLimit*/)
	validProposerSlashings := make([]*ethpb.ProposerSlashing, 0, len(proposerSlashings))
	for _, slashing := range proposerSlashings {
		_, err := blocks.ProcessProposerSlashing(ctx, head, slashing, v.SlashValidator)
		if err != nil {
			log.WithError(err).Warn("Proposer: invalid proposer slashing")
			continue
		}
		validProposerSlashings = append(validProposerSlashings, slashing)
	}

	attSlashings := vs.SlashingsPool.PendingAttesterSlashings(ctx, head, false /*noLimit*/)
	validAttSlashings := make([]*ethpb.AttesterSlashing, 0, len(attSlashings))
	for _, slashing := range attSlashings {
		_, err := blocks.ProcessAttesterSlashing(ctx, head, slashing, v.SlashValidator)
		if err != nil {
			log.WithError(err).Warn("Proposer: invalid attester slashing")
			continue
		}
		validAttSlashings = append(validAttSlashings, slashing)
	}
	exits := vs.ExitPool.PendingExits(head, req.Slot, false /*noLimit*/)
	validExits := make([]*ethpb.SignedVoluntaryExit, 0, len(exits))
	for _, exit := range exits {
		val, err := head.ValidatorAtIndexReadOnly(exit.Exit.ValidatorIndex)
		if err != nil {
			log.WithError(err).Warn("Proposer: invalid exit")
			continue
		}
		if err := blocks.VerifyExitAndSignature(val, head.Slot(), head.Fork(), exit, head.GenesisValidatorsRoot()); err != nil {
			log.WithError(err).Warn("Proposer: invalid exit")
			continue
		}
		validExits = append(validExits, exit)
	}

	blk := &blockData{
		Slot:              req.Slot,
		ParentRoot:        parentRoot,
		Graffiti:          graffiti,
		ProposerIdx:       idx,
		Eth1Data:          eth1Data,
		Deposits:          deposits,
		Attestations:      atts,
		RandaoReveal:      req.RandaoReveal,
		ProposerSlashings: validProposerSlashings,
		AttesterSlashings: validAttSlashings,
		VoluntaryExits:    validExits,
	}

	if slots.ToEpoch(req.Slot) >= params.BeaconConfig().AltairForkEpoch {
		syncAggregate, err := vs.getSyncAggregate(ctx, req.Slot-1, bytesutil.ToBytes32(parentRoot))
		if err != nil {
			return nil, errors.Wrap(err, "could not compute the sync aggregate")
		}

		blk.SyncAggregate = syncAggregate
	}

	if slots.ToEpoch(req.Slot) >= params.BeaconConfig().BellatrixForkEpoch {
		// We request the execution payload only if the validator is not registered with a relayer
		registered, err := vs.validatorRegistered(ctx, idx)
		if !registered || err != nil {
			executionData, err := vs.getExecutionPayload(ctx, req.Slot, idx, bytesutil.ToBytes32(parentRoot), head)
			if err != nil {
				return nil, errors.Wrap(err, "could not get execution payload")
			}
			if slots.ToEpoch(req.Slot) >= params.BeaconConfig().CapellaForkEpoch {
				executionData, blobsBundle, err := vs.getExecutionPayloadV2AndBlobsBundleV1(
					ctx,
					req.Slot,
					idx,
					bytesutil.ToBytes32(parentRoot),
					head,
				)
				if err != nil {
					return nil, errors.Wrap(err, "could not get execution payload")
				}
				p, err := executionData.PbV2()
				if err != nil {
					return nil, errors.Wrap(err, "could not get execution payload v2")
				}

				blk.ExecutionPayloadV2 = p

				blk.BlobsKzg = blobsBundle.KzgCommitments
				aggregatedProof, err := eth.ComputeAggregateKZGProof(blobs.BlobsSequenceImpl(blobsBundle.Blobs))
				if err != nil {
					return nil, fmt.Errorf("failed to compute aggregated kzg proof: %v", err)
				}
				vs.BlobsCache.Put(&ethpb.BlobsSidecar{
					BeaconBlockRoot: blobsBundle.BlockHash,
					BeaconBlockSlot: blk.Slot,
					Blobs:           blobsBundle.Blobs,
					AggregatedProof: aggregatedProof[:],
				})

				changes, err := vs.BLSChangesPool.BLSToExecChangesForInclusion()
				if err != nil {
					return nil, errors.Wrap(err, "could not pack BLSToExecutionChanges")
				}
				blk.BlsToExecutionChanges = changes
			} else {
				p, err := executionData.PbV1()
				if err != nil {
					return nil, errors.Wrap(err, "could not get execution payload v2")
				}
				blk.ExecutionPayload = p
			}
		}
	}

	return blk, nil
}

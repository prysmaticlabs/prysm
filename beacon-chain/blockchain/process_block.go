package blockchain

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	coreTime "github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/monitoring/tracing"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// A custom slot deadline for processing state slots in our cache.
const slotDeadline = 5 * time.Second

// A custom deadline for deposit trie insertion.
const depositDeadline = 20 * time.Second

// This defines size of the upper bound for initial sync block cache.
var initialSyncBlockCacheSize = uint64(2 * params.BeaconConfig().SlotsPerEpoch)

// onBlock is called when a gossip block is received. It runs regular state transition on the block.
// The block's signing root should be computed before calling this method to avoid redundant
// computation in this method and methods it calls into.
//
// Spec pseudocode definition:
//   def on_block(store: Store, signed_block: SignedBeaconBlock) -> None:
//    block = signed_block.message
//    # Parent block must be known
//    assert block.parent_root in store.block_states
//    # Make a copy of the state to avoid mutability issues
//    pre_state = copy(store.block_states[block.parent_root])
//    # Blocks cannot be in the future. If they are, their consideration must be delayed until the are in the past.
//    assert get_current_slot(store) >= block.slot
//
//    # Check that block is later than the finalized epoch slot (optimization to reduce calls to get_ancestor)
//    finalized_slot = compute_start_slot_at_epoch(store.finalized_checkpoint.epoch)
//    assert block.slot > finalized_slot
//    # Check block is a descendant of the finalized block at the checkpoint finalized slot
//    assert get_ancestor(store, block.parent_root, finalized_slot) == store.finalized_checkpoint.root
//
//    # Check the block is valid and compute the post-state
//    state = pre_state.copy()
//    state_transition(state, signed_block, True)
//    # Add new block to the store
//    store.blocks[hash_tree_root(block)] = block
//    # Add new state for this block to the store
//    store.block_states[hash_tree_root(block)] = state
//
//    # Update justified checkpoint
//    if state.current_justified_checkpoint.epoch > store.justified_checkpoint.epoch:
//        if state.current_justified_checkpoint.epoch > store.best_justified_checkpoint.epoch:
//            store.best_justified_checkpoint = state.current_justified_checkpoint
//        if should_update_justified_checkpoint(store, state.current_justified_checkpoint):
//            store.justified_checkpoint = state.current_justified_checkpoint
//
//    # Update finalized checkpoint
//    if state.finalized_checkpoint.epoch > store.finalized_checkpoint.epoch:
//        store.finalized_checkpoint = state.finalized_checkpoint
//
//        # Potentially update justified if different from store
//        if store.justified_checkpoint != state.current_justified_checkpoint:
//            # Update justified if new justified is later than store justified
//            if state.current_justified_checkpoint.epoch > store.justified_checkpoint.epoch:
//                store.justified_checkpoint = state.current_justified_checkpoint
//                return
//
//            # Update justified if store justified is not in chain with finalized checkpoint
//            finalized_slot = compute_start_slot_at_epoch(store.finalized_checkpoint.epoch)
//            ancestor_at_finalized_slot = get_ancestor(store, store.justified_checkpoint.root, finalized_slot)
//            if ancestor_at_finalized_slot != store.finalized_checkpoint.root:
//                store.justified_checkpoint = state.current_justified_checkpoint
func (s *Service) onBlock(ctx context.Context, signed block.SignedBeaconBlock, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.onBlock")
	defer span.End()
	if err := helpers.BeaconBlockIsNil(signed); err != nil {
		return err
	}
	b := signed.Block()

	preState, err := s.getBlockPreState(ctx, b)
	if err != nil {
		return err
	}
	// TODO_MERGE: Optimize this copy.
	copiedPreState := preState.Copy()

	body := signed.Block().Body()
	// TODO_MERGE: Break `ExecuteStateTransition` into per_slot and block processing so we can call `ExecutePayload` in the middle.
	postState, err := transition.ExecuteStateTransition(ctx, preState, signed)
	if err != nil {
		// TODO_MERGE: Notify execution client in the event of invalid conensus block
		return err
	}

	fullyValidated := false
	if copiedPreState.Version() == version.Bellatrix || postState.Version() == version.Bellatrix || copiedPreState.Version() == version.MiniDankSharding || postState.Version() == version.MiniDankSharding {
		executionEnabled, err := blocks.ExecutionEnabled(postState, body)
		if err != nil {
			return errors.Wrap(err, "could not check if execution is enabled")
		}
		if executionEnabled {
			payload, err := body.ExecutionPayload()
			if err != nil {
				return errors.Wrap(err, "could not get body execution payload")
			}
			// This is not the earliest we can call `ExecutePayload`, see above to do as the soonest we can call is after per_slot processing.
			status, err := s.cfg.ExecutionEngineCaller.NewPayload(ctx, payload)
			if err != nil {
				return err
			}
			log.WithFields(logrus.Fields{
				"status:":    status.Status,
				"hash:":      fmt.Sprintf("%#x", payload.BlockHash),
				"parentHash": fmt.Sprintf("%#x", payload.ParentHash),
			}).Info("Successfully called newPayload")

			switch status.Status {
			case enginev1.PayloadStatus_INVALID, enginev1.PayloadStatus_INVALID_BLOCK_HASH, enginev1.PayloadStatus_INVALID_TERMINAL_BLOCK:
				// TODO_MERGE walk up the parent chain removing
				return fmt.Errorf("could not prcess execution payload with status : %v", status.Status)
			case enginev1.PayloadStatus_SYNCING, enginev1.PayloadStatus_ACCEPTED:
				candidate, err := s.optimisticCandidateBlock(ctx, b)
				if err != nil {
					return errors.Wrap(err, "could not check if block is optimistic candidate")
				}
				if !candidate {
					return errors.New("could not optimistically sync block")
				}
				log.WithFields(logrus.Fields{
					"slot":        b.Slot(),
					"root":        fmt.Sprintf("%#x", bytesutil.Trunc(blockRoot[:])),
					"payloadHash": fmt.Sprintf("%#x", bytesutil.Trunc(payload.BlockHash)),
				}).Info("Block is optimistic candidate")
				break
			case enginev1.PayloadStatus_VALID:
				fullyValidated = true
			default:
				return errors.New("unknown payload status")
			}
			if fullyValidated {
				mergeBlock, err := blocks.MergeTransitionBlock(copiedPreState, body)
				if err != nil {
					return errors.Wrap(err, "could not check if merge block is terminal")
				}
				if mergeBlock {
					if err := s.validateTerminalBlock(ctx, signed); err != nil {
						return err
					}
				}
			}
		}
	}

	// We add a proposer score boost to fork choice for the block root if applicable, right after
	// running a successful state transition for the block.
	if err := s.cfg.ForkChoiceStore.BoostProposerRoot(
		ctx, signed.Block().Slot(), blockRoot, s.genesisTime,
	); err != nil {
		return err
	}

	if err := s.savePostStateInfo(ctx, blockRoot, signed, postState, false /* reg sync */); err != nil {
		return err
	}

	// update forkchoice synced tips if the block is not optimistic
	if postState.Version() == version.MiniDankSharding || postState.Version() == version.Bellatrix || fullyValidated {
		root, err := b.HashTreeRoot()
		if err != nil {
			return err
		}
		if err := s.cfg.ForkChoiceStore.UpdateSyncedTipsWithValidRoot(ctx, root); err != nil {
			return err
		}
		if err := s.saveSyncedTipsDB(ctx); err != nil {
			return err
		}
	}

	// If slasher is configured, forward the attestations in the block via
	// an event feed for processing.
	if features.Get().EnableSlasher {
		// Feed the indexed attestation to slasher if enabled. This action
		// is done in the background to avoid adding more load to this critical code path.
		go func() {
			// Using a different context to prevent timeouts as this operation can be expensive
			// and we want to avoid affecting the critical code path.
			ctx := context.TODO()
			for _, att := range signed.Block().Body().Attestations() {
				committee, err := helpers.BeaconCommitteeFromState(ctx, preState, att.Data.Slot, att.Data.CommitteeIndex)
				if err != nil {
					log.WithError(err).Error("Could not get attestation committee")
					tracing.AnnotateError(span, err)
					return
				}
				indexedAtt, err := attestation.ConvertToIndexed(ctx, att, committee)
				if err != nil {
					log.WithError(err).Error("Could not convert to indexed attestation")
					tracing.AnnotateError(span, err)
					return
				}
				s.cfg.SlasherAttestationsFeed.Send(indexedAtt)
			}
		}()
	}

	// Update justified check point.
	justified := s.store.JustifiedCheckpt()
	if justified == nil {
		return errNilJustifiedInStore
	}
	currJustifiedEpoch := justified.Epoch
	if postState.CurrentJustifiedCheckpoint().Epoch > currJustifiedEpoch {
		if err := s.updateJustified(ctx, postState); err != nil {
			return err
		}
	}

	finalized := s.store.FinalizedCheckpt()
	if finalized == nil {
		return errNilFinalizedInStore
	}
	newFinalized := postState.FinalizedCheckpointEpoch() > finalized.Epoch
	if newFinalized {
		s.store.SetPrevFinalizedCheckpt(finalized)
		s.store.SetFinalizedCheckpt(postState.FinalizedCheckpoint())
		s.store.SetPrevJustifiedCheckpt(justified)
		s.store.SetJustifiedCheckpt(postState.CurrentJustifiedCheckpoint())
	}

	balances, err := s.justifiedBalances.get(ctx, bytesutil.ToBytes32(justified.Root))
	if err != nil {
		msg := fmt.Sprintf("could not read balances for state w/ justified checkpoint %#x", justified.Root)
		return errors.Wrap(err, msg)
	}
	if err := s.updateHead(ctx, balances); err != nil {
		log.WithError(err).Warn("Could not update head")
	}

	// Notify execution layer with fork choice head update if this is post merge block.
	if postState.Version() == version.Bellatrix || postState.Version() == version.MiniDankSharding {
		executionEnabled, err := blocks.ExecutionEnabled(postState, body)
		if err != nil {
			return errors.Wrap(err, "could not check if execution is enabled")
		}
		if executionEnabled {
			headPayload, err := s.headBlock().Block().Body().ExecutionPayload()
			if err != nil {
				return err
			}
			// TODO_MERGE: Loading the finalized block from DB on per block is not ideal. Finalized block should be cached here
			finalizedBlock, err := s.cfg.BeaconDB.Block(ctx, bytesutil.ToBytes32(finalized.Root))
			if err != nil {
				return err
			}
			finalizedBlockHash := params.BeaconConfig().ZeroHash[:]
			if finalizedBlock != nil && (finalizedBlock.Version() == version.Bellatrix || finalizedBlock.Version() == version.MiniDankSharding) {
				finalizedPayload, err := finalizedBlock.Block().Body().ExecutionPayload()
				if err != nil {
					return err
				}
				finalizedBlockHash = finalizedPayload.BlockHash
			}

			fcs := &enginev1.ForkchoiceState{
				HeadBlockHash:      headPayload.BlockHash,
				SafeBlockHash:      headPayload.BlockHash,
				FinalizedBlockHash: finalizedBlockHash,
			}
			resp, err := s.cfg.ExecutionEngineCaller.ForkchoiceUpdated(ctx, fcs, nil /* attribute */)
			if err != nil {
				return err
			}

			log.WithFields(logrus.Fields{
				"status:": resp.Status.Status,
				"hash:":   fmt.Sprintf("%#x", headPayload.BlockHash),
			}).Info("Successfully called forkchoiceUpdated")

			switch resp.Status.Status {
			case enginev1.PayloadStatus_INVALID, enginev1.PayloadStatus_INVALID_BLOCK_HASH, enginev1.PayloadStatus_INVALID_TERMINAL_BLOCK:
				return fmt.Errorf("could not prcess execution payload with status : %v", resp.Status.Status)
			case enginev1.PayloadStatus_SYNCING, enginev1.PayloadStatus_ACCEPTED:
				candidate, err := s.optimisticCandidateBlock(ctx, b)
				if err != nil {
					return errors.Wrap(err, "could not check if block is optimistic candidate")
				}
				if !candidate {
					return errors.Wrap(err, "could not optimistically sync block")
				}
				break
			case enginev1.PayloadStatus_VALID:
			default:
				return errors.Wrap(err, "could not execute payload")
			}
		}
	}

	if err := s.pruneCanonicalAttsFromPool(ctx, blockRoot, signed); err != nil {
		return err
	}

	// Send notification of the processed block to the state feed.
	s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			Slot:        signed.Block().Slot(),
			BlockRoot:   blockRoot,
			SignedBlock: signed,
			Verified:    true,
		},
	})

	// Updating next slot state cache can happen in the background. It shouldn't block rest of the process.
	go func() {
		// Use a custom deadline here, since this method runs asynchronously.
		// We ignore the parent method's context and instead create a new one
		// with a custom deadline, therefore using the background context instead.
		slotCtx, cancel := context.WithTimeout(context.Background(), slotDeadline)
		defer cancel()
		if err := transition.UpdateNextSlotCache(slotCtx, blockRoot[:], postState); err != nil {
			log.WithError(err).Debug("could not update next slot state cache")
		}
	}()

	// Save justified check point to db.
	if postState.CurrentJustifiedCheckpoint().Epoch > currJustifiedEpoch {
		if err := s.cfg.BeaconDB.SaveJustifiedCheckpoint(ctx, postState.CurrentJustifiedCheckpoint()); err != nil {
			return err
		}
	}

	// Update finalized check point.
	if newFinalized {
		if err := s.updateFinalized(ctx, postState.FinalizedCheckpoint()); err != nil {
			return err
		}
		fRoot := bytesutil.ToBytes32(postState.FinalizedCheckpoint().Root)
		if err := s.cfg.ForkChoiceStore.Prune(ctx, fRoot); err != nil {
			return errors.Wrap(err, "could not prune proto array fork choice nodes")
		}
		go func() {
			// Send an event regarding the new finalized checkpoint over a common event feed.
			s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
				Type: statefeed.FinalizedCheckpoint,
				Data: &ethpbv1.EventFinalizedCheckpoint{
					Epoch: postState.FinalizedCheckpoint().Epoch,
					Block: postState.FinalizedCheckpoint().Root,
					State: signed.Block().StateRoot(),
				},
			})

			// Use a custom deadline here, since this method runs asynchronously.
			// We ignore the parent method's context and instead create a new one
			// with a custom deadline, therefore using the background context instead.
			depCtx, cancel := context.WithTimeout(context.Background(), depositDeadline)
			defer cancel()
			if err := s.insertFinalizedDeposits(depCtx, fRoot); err != nil {
				log.WithError(err).Error("Could not insert finalized deposits.")
			}
		}()

	}

	defer reportAttestationInclusion(b)

	return s.handleEpochBoundary(ctx, postState)
}

func (s *Service) onBlockBatch(ctx context.Context, blks []block.SignedBeaconBlock,
	blockRoots [][32]byte) ([]*ethpb.Checkpoint, []*ethpb.Checkpoint, []bool, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.onBlockBatch")
	defer span.End()

	if len(blks) == 0 || len(blockRoots) == 0 {
		return nil, nil, nil, errors.New("no blocks provided")
	}
	if err := helpers.BeaconBlockIsNil(blks[0]); err != nil {
		return nil, nil, nil, err
	}
	b := blks[0].Block()

	// Retrieve incoming block's pre state.
	if err := s.verifyBlkPreState(ctx, b); err != nil {
		return nil, nil, nil, err
	}
	preState, err := s.cfg.StateGen.StateByRootInitialSync(ctx, bytesutil.ToBytes32(b.ParentRoot()))
	if err != nil {
		return nil, nil, nil, err
	}
	if preState == nil || preState.IsNil() {
		return nil, nil, nil, fmt.Errorf("nil pre state for slot %d", b.Slot())
	}

	jCheckpoints := make([]*ethpb.Checkpoint, len(blks))
	fCheckpoints := make([]*ethpb.Checkpoint, len(blks))
	optimistic := make([]bool, len(blks))
	sigSet := &bls.SignatureBatch{
		Signatures: [][]byte{},
		PublicKeys: []bls.PublicKey{},
		Messages:   [][32]byte{},
	}
	var set *bls.SignatureBatch
	boundaries := make(map[[32]byte]state.BeaconState)
	for i, b := range blks {
		preStateCopied := preState.Copy() // TODO_MERGE: Optimize this copy.
		set, preState, err = transition.ExecuteStateTransitionNoVerifyAnySig(ctx, preState, b)
		if err != nil {
			return nil, nil, nil, err
		}

		// Non merge blocks are never optimistic
		optimistic[i] = false
		if preState.Version() == version.Bellatrix {
			executionEnabled, err := blocks.ExecutionEnabled(preState, b.Block().Body())
			if err != nil {
				return nil, nil, nil, errors.Wrap(err, "could not check if execution is enabled")
			}
			if executionEnabled {
				payload, err := b.Block().Body().ExecutionPayload()
				if err != nil {
					return nil, nil, nil, errors.Wrap(err, "could not get body execution payload")
				}
				status, err := s.cfg.ExecutionEngineCaller.NewPayload(ctx, payload)
				if err != nil {
					return nil, nil, nil, err
				}
				switch status.Status {
				case enginev1.PayloadStatus_INVALID, enginev1.PayloadStatus_INVALID_BLOCK_HASH, enginev1.PayloadStatus_INVALID_TERMINAL_BLOCK:
					// TODO_MERGE walk up the parent chain removing
					return nil, nil, nil, fmt.Errorf("could not prcess execution payload with status : %v", status.Status)
				case enginev1.PayloadStatus_SYNCING, enginev1.PayloadStatus_ACCEPTED:
					candidate, err := s.optimisticCandidateBlock(ctx, b.Block())
					if err != nil {
						return nil, nil, nil, errors.Wrap(err, "could not check if block is optimistic candidate")
					}
					if !candidate {
						return nil, nil, nil, errors.New("could not optimistically sync block")
					}
					log.WithFields(logrus.Fields{
						"slot":        b.Block().Slot(),
						"root":        fmt.Sprintf("%#x", bytesutil.Trunc(blockRoots[i][:])),
						"payloadHash": fmt.Sprintf("%#x", bytesutil.Trunc(payload.BlockHash)),
					}).Info("Block is optimistic candidate")
					optimistic[i] = true
					break
				case enginev1.PayloadStatus_VALID:
				default:
					return nil, nil, nil, errors.New("unknown payload status")
				}
				if !optimistic[i] {
					mergeBlock, err := blocks.MergeTransitionBlock(preStateCopied, b.Block().Body())
					if err != nil {
						return nil, nil, nil, errors.Wrap(err, "could not check if merge block is terminal")
					}
					if mergeBlock {
						if err := s.validateTerminalBlock(ctx, b); err != nil {
							return nil, nil, nil, err
						}
					}
				}

				headPayload, err := b.Block().Body().ExecutionPayload()
				if err != nil {
					return nil, nil, nil, err

				}
				// TODO_MERGE: Loading the finalized block from DB on per block is not ideal. Finalized block should be cached here
				finalizedBlock, err := s.cfg.BeaconDB.Block(ctx, bytesutil.ToBytes32(preState.FinalizedCheckpoint().Root))
				if err != nil {
					return nil, nil, nil, err

				}
				finalizedBlockHash := params.BeaconConfig().ZeroHash[:]
				if finalizedBlock != nil && finalizedBlock.Version() == version.Bellatrix {
					finalizedPayload, err := finalizedBlock.Block().Body().ExecutionPayload()
					if err != nil {
						return nil, nil, nil, err

					}
					finalizedBlockHash = finalizedPayload.BlockHash
				}

				fcs := &enginev1.ForkchoiceState{
					HeadBlockHash:      headPayload.BlockHash,
					SafeBlockHash:      headPayload.BlockHash,
					FinalizedBlockHash: finalizedBlockHash,
				}

				resp, err := s.cfg.ExecutionEngineCaller.ForkchoiceUpdated(ctx, fcs, nil /* attribute */)
				if err != nil {
					return nil, nil, nil, err
				}
				switch resp.Status.Status {
				case enginev1.PayloadStatus_INVALID, enginev1.PayloadStatus_INVALID_BLOCK_HASH, enginev1.PayloadStatus_INVALID_TERMINAL_BLOCK:
					return nil, nil, nil, fmt.Errorf("could not prcess execution payload with status : %v", resp.Status.Status)
				case enginev1.PayloadStatus_SYNCING, enginev1.PayloadStatus_ACCEPTED:
					break
				case enginev1.PayloadStatus_VALID:
				default:
					return nil, nil, nil, errors.Wrap(err, "could not execute payload")
				}
			}
		}

		// Save potential boundary states.
		if slots.IsEpochStart(preState.Slot()) {
			boundaries[blockRoots[i]] = preState.Copy()
			if err := s.handleEpochBoundary(ctx, preState); err != nil {
				return nil, nil, nil, errors.Wrap(err, "could not handle epoch boundary state")
			}
		}
		jCheckpoints[i] = preState.CurrentJustifiedCheckpoint()
		fCheckpoints[i] = preState.FinalizedCheckpoint()

		sigSet.Join(set)
	}
	verify, err := sigSet.Verify()
	if err != nil {
		return nil, nil, nil, err
	}
	if !verify {
		return nil, nil, nil, errors.New("batch block signature verification failed")
	}
	for r, st := range boundaries {
		if err := s.cfg.StateGen.SaveState(ctx, r, st); err != nil {
			return nil, nil, nil, err
		}
	}
	// Also saves the last post state which to be used as pre state for the next batch.
	lastB := blks[len(blks)-1]
	lastBR := blockRoots[len(blockRoots)-1]
	if err := s.cfg.StateGen.SaveState(ctx, lastBR, preState); err != nil {
		return nil, nil, nil, err
	}
	if err := s.saveHeadNoDB(ctx, lastB, lastBR, preState); err != nil {
		return nil, nil, nil, err
	}
	return fCheckpoints, jCheckpoints, optimistic, nil
}

// handles a block after the block's batch has been verified, where we can save blocks
// their state summaries and split them off to relative hot/cold storage.
func (s *Service) handleBlockAfterBatchVerify(ctx context.Context, signed block.SignedBeaconBlock,
	blockRoot [32]byte, fCheckpoint, jCheckpoint *ethpb.Checkpoint) error {
	b := signed.Block()

	s.saveInitSyncBlock(blockRoot, signed)
	if err := s.insertBlockToForkChoiceStore(ctx, b, blockRoot, fCheckpoint, jCheckpoint); err != nil {
		return err
	}
	if err := s.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{
		Slot: signed.Block().Slot(),
		Root: blockRoot[:],
	}); err != nil {
		return err
	}

	// Rate limit how many blocks (2 epochs worth of blocks) a node keeps in the memory.
	if uint64(len(s.getInitSyncBlocks())) > initialSyncBlockCacheSize {
		if err := s.cfg.BeaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
			return err
		}
		s.clearInitSyncBlocks()
	}

	justified := s.store.JustifiedCheckpt()
	if justified == nil {
		return errNilJustifiedInStore
	}
	if jCheckpoint.Epoch > justified.Epoch {
		if err := s.updateJustifiedInitSync(ctx, jCheckpoint); err != nil {
			return err
		}
	}

	finalized := s.store.FinalizedCheckpt()
	if finalized == nil {
		return errNilFinalizedInStore
	}
	// Update finalized check point. Prune the block cache and helper caches on every new finalized epoch.
	if fCheckpoint.Epoch > finalized.Epoch {
		if err := s.updateFinalized(ctx, fCheckpoint); err != nil {
			return err
		}
		s.store.SetPrevFinalizedCheckpt(finalized)
		s.store.SetFinalizedCheckpt(fCheckpoint)
	}
	return nil
}

// Epoch boundary bookkeeping such as logging epoch summaries.
func (s *Service) handleEpochBoundary(ctx context.Context, postState state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.handleEpochBoundary")
	defer span.End()

	if postState.Slot()+1 == s.nextEpochBoundarySlot {
		// Update caches for the next epoch at epoch boundary slot - 1.
		if err := helpers.UpdateCommitteeCache(postState, coreTime.NextEpoch(postState)); err != nil {
			return err
		}
		copied := postState.Copy()
		copied, err := transition.ProcessSlots(ctx, copied, copied.Slot()+1)
		if err != nil {
			return err
		}
		if err := helpers.UpdateProposerIndicesInCache(ctx, copied); err != nil {
			return err
		}
	} else if postState.Slot() >= s.nextEpochBoundarySlot {
		if err := reportEpochMetrics(ctx, postState, s.head.state); err != nil {
			return err
		}
		var err error
		s.nextEpochBoundarySlot, err = slots.EpochStart(coreTime.NextEpoch(postState))
		if err != nil {
			return err
		}

		// Update caches at epoch boundary slot.
		// The following updates have short cut to return nil cheaply if fulfilled during boundary slot - 1.
		if err := helpers.UpdateCommitteeCache(postState, coreTime.CurrentEpoch(postState)); err != nil {
			return err
		}
		if err := helpers.UpdateProposerIndicesInCache(ctx, postState); err != nil {
			return err
		}
	}

	return nil
}

// This feeds in the block and block's attestations to fork choice store. It's allows fork choice store
// to gain information on the most current chain.
func (s *Service) insertBlockAndAttestationsToForkChoiceStore(ctx context.Context, blk block.BeaconBlock, root [32]byte,
	st state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.insertBlockAndAttestationsToForkChoiceStore")
	defer span.End()

	fCheckpoint := st.FinalizedCheckpoint()
	jCheckpoint := st.CurrentJustifiedCheckpoint()
	if err := s.insertBlockToForkChoiceStore(ctx, blk, root, fCheckpoint, jCheckpoint); err != nil {
		return err
	}
	// Feed in block's attestations to fork choice store.
	for _, a := range blk.Body().Attestations() {
		committee, err := helpers.BeaconCommitteeFromState(ctx, st, a.Data.Slot, a.Data.CommitteeIndex)
		if err != nil {
			return err
		}
		indices, err := attestation.AttestingIndices(a.AggregationBits, committee)
		if err != nil {
			return err
		}
		s.cfg.ForkChoiceStore.ProcessAttestation(ctx, indices, bytesutil.ToBytes32(a.Data.BeaconBlockRoot), a.Data.Target.Epoch)
	}
	return nil
}

func (s *Service) insertBlockToForkChoiceStore(ctx context.Context, blk block.BeaconBlock,
	root [32]byte, fCheckpoint, jCheckpoint *ethpb.Checkpoint) error {
	if err := s.fillInForkChoiceMissingBlocks(ctx, blk, fCheckpoint, jCheckpoint); err != nil {
		return err
	}
	// Feed in block to fork choice store.
	if err := s.cfg.ForkChoiceStore.ProcessBlock(ctx,
		blk.Slot(), root, bytesutil.ToBytes32(blk.ParentRoot()), bytesutil.ToBytes32(blk.Body().Graffiti()),
		jCheckpoint.Epoch,
		fCheckpoint.Epoch); err != nil {
		return errors.Wrap(err, "could not process block for proto array fork choice")
	}
	return nil
}

// This saves post state info to DB or cache. This also saves post state info to fork choice store.
// Post state info consists of processed block and state. Do not call this method unless the block and state are verified.
func (s *Service) savePostStateInfo(ctx context.Context, r [32]byte, b block.SignedBeaconBlock, st state.BeaconState, initSync bool) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.savePostStateInfo")
	defer span.End()
	if initSync {
		s.saveInitSyncBlock(r, b)
	} else if err := s.cfg.BeaconDB.SaveBlock(ctx, b); err != nil {
		return errors.Wrapf(err, "could not save block from slot %d", b.Block().Slot())
	}
	if err := s.cfg.StateGen.SaveState(ctx, r, st); err != nil {
		return errors.Wrap(err, "could not save state")
	}
	if err := s.insertBlockAndAttestationsToForkChoiceStore(ctx, b.Block(), r, st); err != nil {
		return errors.Wrapf(err, "could not insert block %d to fork choice store", b.Block().Slot())
	}
	return nil
}

// This removes the attestations from the mem pool. It will only remove the attestations if input root `r` is canonical,
// meaning the block `b` is part of the canonical chain.
func (s *Service) pruneCanonicalAttsFromPool(ctx context.Context, r [32]byte, b block.SignedBeaconBlock) error {
	if !features.Get().CorrectlyPruneCanonicalAtts {
		return nil
	}

	canonical, err := s.IsCanonical(ctx, r)
	if err != nil {
		return err
	}
	if !canonical {
		return nil
	}

	atts := b.Block().Body().Attestations()
	for _, att := range atts {
		if helpers.IsAggregated(att) {
			if err := s.cfg.AttPool.DeleteAggregatedAttestation(att); err != nil {
				return err
			}
		} else {
			if err := s.cfg.AttPool.DeleteUnaggregatedAttestation(att); err != nil {
				return err
			}
		}
	}
	return nil
}

// validates terminal block hash in the event of manual overrides before checking for total difficulty.
//
// def validate_merge_block(block: BeaconBlock) -> None:
//    """
//    Check the parent PoW block of execution payload is a valid terminal PoW block.
//
//    Note: Unavailable PoW block(s) may later become available,
//    and a client software MAY delay a call to ``validate_merge_block``
//    until the PoW block(s) become available.
//    """
//    if TERMINAL_BLOCK_HASH != Hash32():
//        # If `TERMINAL_BLOCK_HASH` is used as an override, the activation epoch must be reached.
//        assert compute_epoch_at_slot(block.slot) >= TERMINAL_BLOCK_HASH_ACTIVATION_EPOCH
//        return block.block_hash == TERMINAL_BLOCK_HASH
//
//    pow_block = get_pow_block(block.body.execution_payload.parent_hash)
//    # Check if `pow_block` is available
//    assert pow_block is not None
//    pow_parent = get_pow_block(pow_block.parent_hash)
//    # Check if `pow_parent` is available
//    assert pow_parent is not None
//    # Check if `pow_block` is a valid terminal PoW block
//    assert is_valid_terminal_pow_block(pow_block, pow_parent)
func (s *Service) validateTerminalBlock(ctx context.Context, b block.SignedBeaconBlock) error {
	payload, err := b.Block().Body().ExecutionPayload()
	if err != nil {
		return err
	}
	if bytesutil.ToBytes32(params.BeaconConfig().TerminalBlockHash.Bytes()) != [32]byte{} {
		// `TERMINAL_BLOCK_HASH` is used as an override, the activation epoch must be reached.
		if params.BeaconConfig().TerminalBlockHashActivationEpoch > slots.ToEpoch(b.Block().Slot()) {
			return errors.New("terminal block hash activation epoch not reached")
		}
		if !bytes.Equal(payload.ParentHash, params.BeaconConfig().TerminalBlockHash.Bytes()) {
			return errors.New("parent hash does not match terminal block hash")
		}
		return nil
	}
	transitionBlk, err := s.cfg.ExecutionEngineCaller.ExecutionBlockByHash(ctx, common.BytesToHash(payload.ParentHash))
	if err != nil {
		return errors.Wrap(err, "could not get transition block")
	}
	parentTransitionBlk, err := s.cfg.ExecutionEngineCaller.ExecutionBlockByHash(ctx, common.BytesToHash(transitionBlk.ParentHash))
	if err != nil {
		return errors.Wrap(err, "could not get transition parent block")
	}
	transitionBlkTDBig, err := hexutil.DecodeBig(transitionBlk.TotalDifficulty)
	if err != nil {
		return errors.Wrap(err, "could not decode transition total difficulty")
	}
	transitionBlkTTD, overflows := uint256.FromBig(transitionBlkTDBig)
	if overflows {
		return errors.New("total difficulty overflows")
	}
	parentBlkTD, err := hexutil.DecodeBig(parentTransitionBlk.TotalDifficulty)
	if err != nil {
		return errors.Wrap(err, "could not decode transition total difficulty")
	}
	parentBlkTTD, overflows := uint256.FromBig(parentBlkTD)
	if overflows {
		return errors.New("total difficulty overflows")
	}
	log.WithFields(logrus.Fields{
		"slot":                                 b.Block().Slot(),
		"transitionBlockHash":                  common.BytesToHash(payload.ParentHash).String(),
		"transitionBlockParentHash":            common.BytesToHash(transitionBlk.ParentHash).String(),
		"terminalTotalDifficulty":              params.BeaconConfig().TerminalTotalDifficulty,
		"transitionBlockTotalDifficulty":       transitionBlkTTD,
		"transitionBlockParentTotalDifficulty": parentBlkTTD,
	}).Info("Validating terminal block")

	validated, err := validTerminalPowBlock(transitionBlkTTD, parentBlkTTD)
	if err != nil {
		return err
	}
	if !validated {
		return errors.New("invalid difficulty for terminal block")
	}

	return nil
}

// Saves synced and validated tips to DB.
func (s *Service) saveSyncedTipsDB(ctx context.Context) error {
	tips := s.cfg.ForkChoiceStore.SyncedTips()
	if len(tips) == 0 {
		return nil
	}
	return s.cfg.BeaconDB.UpdateValidatedTips(ctx, tips)
}

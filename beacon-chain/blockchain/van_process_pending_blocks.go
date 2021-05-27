package blockchain

import (
	"context"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/shared/event"
	vanTypes "github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
	"sort"
	"time"
)

type OrcClient interface {
	ConfirmVanBlockHashes(ctx context.Context, request []*vanTypes.ConfirmationReqData) (response []*vanTypes.ConfirmationResData, err error)
}

var (
	// Getting confirmation status from orchestrator after each confirmationStatusFetchingInverval
	confirmationStatusFetchingInverval = 1 * time.Second
	// maxPendingBlockTryLimit is the maximum limit for pending status of a block
	maxPendingBlockTryLimit       = 5
	errInvalidBlock               = errors.New("invalid block found, discarded block batch")
	errPendingBlockCtxIsDone      = errors.New("pending block confirmation context is done, reinitialize")
	errEmptyBlocksBatch           = errors.New("empty length of the batch of incoming blocks")
	errPendingBlockTryLimitExceed = errors.New("maximum wait is exceeded and orchestrator can not verify the block")
	errUnknownStatus              = errors.New("invalid status from orchestrator")
)

type blockRoot [32]byte

// PendingBlocksFetcher retrieves the cached un-confirmed beacon blocks from cache
type PendingBlocksFetcher interface {
	SortedUnConfirmedBlocksFromCache() ([]*ethpb.BeaconBlock, error)
}

// publishAndStorePendingBlock method publishes and stores the pending block for final confirmation check
func (s *Service) publishAndStorePendingBlock(ctx context.Context, pendingBlk *ethpb.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.publishAndStorePendingBlock")
	defer span.End()

	// Sending pending block feed to streaming api
	log.WithField("slot", pendingBlk.Slot).Debug("Unconfirmed block sends for publishing")
	s.blockNotifier.BlockFeed().Send(&feed.Event{
		Type: blockfeed.UnConfirmedBlock,
		Data: &blockfeed.UnConfirmedBlockData{Block: pendingBlk},
	})

	// Storing pending block into pendingBlockCache
	if err := s.pendingBlockCache.AddPendingBlock(pendingBlk); err != nil {
		return errors.Wrapf(err, "could not cache block of slot %d", pendingBlk.Slot)
	}

	return nil
}

// publishAndStorePendingBlockBatch method publishes and stores the batch of pending block for final confirmation check
// Should get sorted pendingBlkBatch
func (s *Service) publishAndStorePendingBlockBatch(ctx context.Context, pendingBlkBatch []*ethpb.SignedBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.publishAndStorePendingBlockBatch")
	defer span.End()

	for _, b := range pendingBlkBatch {
		// Sending pending block feed to streaming api
		log.WithField("slot", b.Block.Slot).Debug("Unconfirmed block batch sends for publishing")
		s.blockNotifier.BlockFeed().Send(&feed.Event{
			Type: blockfeed.UnConfirmedBlock,
			Data: &blockfeed.UnConfirmedBlockData{Block: b.Block},
		})

		// Storing pending block into pendingBlockCache
		if err := s.pendingBlockCache.AddPendingBlock(b.Block); err != nil {
			return errors.Wrapf(err, "could not cache block of slot %d", b.Block.Slot)
		}
	}

	return nil
}

// UnConfirmedBlocksFromCache retrieves all the cached blocks from cache and send it back to event api
func (s *Service) SortedUnConfirmedBlocksFromCache() ([]*ethpb.BeaconBlock, error) {
	blks, err := s.pendingBlockCache.PendingBlocks()
	if err != nil {
		return nil, errors.Wrap(err, "Could not retrieve cached unconfirmed blocks from cache")
	}

	sort.Slice(blks, func(i, j int) bool {
		return blks[i].Slot < blks[j].Slot
	})

	return blks, nil
}

// processOrcConfirmation runs every certain interval and fetch confirmation from orchestrator periodically and
// publish the confirmation status to its subscriber methods. This loop will run in separate go routine when blockchain
// service starts.
func (s *Service) processOrcConfirmationLoop(ctx context.Context) {
	ticker := time.NewTicker(confirmationStatusFetchingInverval)
	go func() {
		for {
			select {
			case <-ticker.C:
				log.WithField("function", "processOrcConfirmation").Trace("running")
				if err := s.fetchConfirmations(ctx); err != nil {
					log.WithError(err).Error("got error when calling fetchOrcConfirmations method. exiting!")
					return
				}
				continue
			case <-ctx.Done():
				log.WithField("function", "processOrcConfirmation").Debug("context is closed, exiting")
				ticker.Stop()
				return
			}
		}
	}()
}

// fetchOrcConfirmations process confirmation for pending blocks
// -> After getting confirmation for a list of pending slots, it iterates through the list
// -> If any slot gets invalid status then stop the iteration and start again from that slot
// -> If any slot gets verified status then, publish the slots and block hashes to the blockchain service
//    who actually waiting for confirmed blocks
// -> If any slot gets un
func (s *Service) fetchConfirmations(ctx context.Context) error {
	reqData, err := s.sortedPendingSlots()
	if err != nil {
		log.WithError(err).Error("got error when preparing sorted confirmation request data")
		return err
	}

	if len(reqData) == 0 {
		return nil
	}

	if s.orcRPCClient == nil {
		return nil
	}

	resData, err := s.orcRPCClient.ConfirmVanBlockHashes(ctx, reqData)
	if err != nil {
		log.WithError(err).Error("got error when fetching confirmations from orchestrator")
		return err
	}

	for i := 0; i < len(resData); i++ {
		log.WithField("slot", resData[i].Slot).WithField(
			"status", resData[i].Status).Debug("got confirmation status from orchestrator")

		s.blockNotifier.BlockFeed().Send(&feed.Event{
			Type: blockfeed.ConfirmedBlock,
			Data: &blockfeed.ConfirmedData{
				Slot:          resData[i].Slot,
				BlockRootHash: resData[i].Hash,
				Status:        resData[i].Status,
			},
		})
	}

	return nil
}

// waitForConfirmationBlock method gets a block. It gets the status using notification by processOrcConfirmationLoop and then it
// takes action based on status of block. If status is-
// Verified:
// 	- return nil
// Invalid:
//	- directly return error and discard the pending block.
//	- sync service will re-download the block
// Pending:
//	- Re-check new response from orchestrator
//  - Decrease the re-try limit if it gets pending status again
//	- If it reaches the maximum limit then return error
func (s *Service) waitForConfirmationBlock(ctx context.Context, b *ethpb.SignedBeaconBlock) error {
	confirmedBlocksCh := make(chan *feed.Event, 1)
	var confirmedBlockSub event.Subscription

	confirmedBlockSub = s.blockNotifier.BlockFeed().Subscribe(confirmedBlocksCh)
	defer confirmedBlockSub.Unsubscribe()
	pendingBlockTryLimit := maxPendingBlockTryLimit

	for {
		select {
		case statusData := <-confirmedBlocksCh:
			if statusData.Type == blockfeed.ConfirmedBlock {
				data, ok := statusData.Data.(*blockfeed.ConfirmedData)
				if !ok || data == nil {
					continue
				}
				// Checks slot number with incoming confirmation data slot
				if data.Slot == b.Block.Slot {
					switch status := data.Status; status {
					case vanTypes.Verified:
						log.WithField("slot", data.Slot).WithField(
							"blockHash", data.BlockRootHash).Debug(
							"got verified status from orchestrator")
						if err := s.pendingBlockCache.DeleteConfirmedBlock(b.Block.Slot); err != nil {
							log.WithError(err).Error("couldn't delete the verified blocks from cache")
							return err
						}
						return nil
					case vanTypes.Pending:
						log.WithField("slot", data.Slot).WithField(
							"blockHash", data.BlockRootHash).Debug(
							"got pending status from orchestrator, exiting goroutine")

						pendingBlockTryLimit = pendingBlockTryLimit - 1
						if pendingBlockTryLimit == 0 {
							log.WithError(errPendingBlockTryLimitExceed).Error(
								"orchestrator sends pending status for this block so many times")
							return errPendingBlockTryLimitExceed
						}
						continue
					case vanTypes.Invalid:
						log.WithField("slot", data.Slot).WithField(
							"blockHash", data.BlockRootHash).Debug(
							"got invalid status from orchestrator, exiting goroutine")
						return errInvalidBlock
					default:
						log.WithError(errUnknownStatus).Error(
							"got unknown status from orchestrator, exiting goroutine")
						return errUnknownStatus
					}
				}
			}
		case err := <-confirmedBlockSub.Err():
			log.WithError(err).Error("Confirmation fetcher closed, exiting goroutine")
			return err
		case <-s.ctx.Done():
			log.WithField("function",
				"waitForConfirmationBlock").Debug("context is closed, exiting")
			return errPendingBlockCtxIsDone
		}
	}
}

// waitForConfirmationsBlockBatch method gets batch of blocks. It is notified by processOrcConfirmationLoop and then it
// takes action based on status of block. If status is Verified then it will switch to next block. If status
// Verified:
// 	- switch to next block
//	- check if the current verified slot is last one then it return nil error
// Invalid:
//	- directly return error and discard the whole batch.
//	- sync service will re-download the batch
// Pending:
//	- Re-check new response from orchestrator
//  - Decrease the re-try limit if it gets pending status again
//	- If it reaches the maximum limit then return error
func (s *Service) waitForConfirmationsBlockBatch(ctx context.Context, blocks []*ethpb.SignedBeaconBlock) error {
	if len(blocks) <= 0 {
		log.WithError(errEmptyBlocksBatch).Error("incoming batch length is zero")
		return errEmptyBlocksBatch
	}

	confirmedBlocksCh := make(chan *feed.Event, 1)
	var confirmedBlockSub event.Subscription
	confirmedBlockSub = s.blockNotifier.BlockFeed().Subscribe(confirmedBlocksCh)
	defer confirmedBlockSub.Unsubscribe()

	curVerifiedSlot := types.Slot(0)
	slotCounter := 0
	nextSlotToVerify := blocks[slotCounter].Block.Slot
	lastSlot := blocks[len(blocks)-1].Block.Slot
	pendingBlockTryLimit := maxPendingBlockTryLimit

	for {
		select {
		case statusData := <-confirmedBlocksCh:
			if statusData.Type == blockfeed.ConfirmedBlock {
				data, ok := statusData.Data.(*blockfeed.ConfirmedData)
				if !ok || data == nil {
					continue
				}

				// When the published status is only the status for next slot to verify then we check the status
				if nextSlotToVerify == data.Slot {
					switch status := data.Status; status {
					case vanTypes.Verified:
						log.WithField("slot", data.Slot).WithField(
							"blockHash", data.BlockRootHash).Debug("verified by orchestrator")
						curVerifiedSlot = data.Slot
						if curVerifiedSlot == lastSlot {
							for _, b := range blocks {
								if err := s.pendingBlockCache.DeleteConfirmedBlock(b.Block.Slot); err != nil {
									log.WithError(err).Error("couldn't delete the verified blocks from cache")
									return err
								}
							}
							return nil
						}
						slotCounter++
						nextSlotToVerify = blocks[slotCounter].Block.GetSlot()
						// Reset this limit for next pending block
						pendingBlockTryLimit = maxPendingBlockTryLimit
						continue
					case vanTypes.Invalid:
						log.WithField("slot", data.Slot).WithField(
							"blockHash", data.BlockRootHash).Debug(
							"invalid by orchestrator, exiting goroutine")
						return errInvalidBlock
					case vanTypes.Pending:
						log.WithField("slot", data.Slot).WithField(
							"blockHash", data.BlockRootHash).Debug("got pending status from orchestrator")
						pendingBlockTryLimit = pendingBlockTryLimit - 1
						if pendingBlockTryLimit == 0 {
							log.WithError(errPendingBlockTryLimitExceed).Error(
								"orchestrator sends pending status for this block so many times")
							return errPendingBlockTryLimitExceed
						}
						continue
					default:
						log.WithError(errUnknownStatus).Error(
							"got unknown status from orchestrator, exiting goroutine")
						return errUnknownStatus
					}
				}
			}
		case err := <-confirmedBlockSub.Err():
			log.WithError(err).Error("Confirmation fetcher closed, exiting goroutine")
			return err
		case <-s.ctx.Done():
			log.WithField("function", "waitForConfirmationsBlockBatch").Debug("context is closed, exiting")
			return errPendingBlockCtxIsDone
		}
	}
}

// sortedPendingSlots retrieves pending blocks from pending block cache and prepare sorted request data
func (s *Service) sortedPendingSlots() ([]*vanTypes.ConfirmationReqData, error) {
	items, err := s.pendingBlockCache.PendingBlocks()
	if err != nil {
		return nil, err
	}

	reqData := make([]*vanTypes.ConfirmationReqData, 0, len(items))
	for _, blk := range items {
		blockRoot, err := blk.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		reqData = append(reqData, &vanTypes.ConfirmationReqData{
			Slot: blk.Slot,
			Hash: blockRoot,
		})
	}

	sort.Slice(reqData, func(i, j int) bool {
		return reqData[i].Slot < reqData[j].Slot
	})

	return reqData, nil
}

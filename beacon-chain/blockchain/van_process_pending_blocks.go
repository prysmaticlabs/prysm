package blockchain

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/shared/event"
	vanTypes "github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
	"sort"
	"time"
)

var (
	// Getting confirmation status from orchestrator after each confirmationStatusFetchingInverval
	confirmationStatusFetchingInverval = 2 * time.Second
	// maxPendingBlockTryLimit is the maximum limit for pending status of a block
	maxPendingBlockTryLimit       = 10
	errInvalidBlock               = errors.New("invalid block found, discarded block batch")
	errPendingBlockCtxIsDone      = errors.New("pending block confirmation context is done, reinitialize")
	errEmptyBlocksBatch           = errors.New("empty length of the batch of incoming blocks")
	errPendingBlockTryLimitExceed = errors.New("maximum wait is exceeded and orchestrator can not verify the block")
	errUnknownStatus              = errors.New("invalid status from orchestrator")
	errInvalidRPCClient           = errors.New("invalid orchestrator rpc client or no client initiated")
	errSkippedStatus              = errors.New("skipped status from orchestrator")
)

// PendingBlocksFetcher retrieves the cached un-confirmed beacon blocks from cache
type PendingBlocksFetcher interface {
	SortedUnConfirmedBlocksFromCache() ([]*ethpb.BeaconBlock, error)
}

type blockRoot [32]byte

// publishAndWaitForOrcConfirmation publish the block to orchestrator and store the block into pending queue cache
func (s *Service) publishAndWaitForOrcConfirmation(ctx context.Context, pendingBlk *ethpb.SignedBeaconBlock) error {
	// Send the incoming block acknowledge to orchestrator and store the pending block to cache
	if err := s.publishAndStorePendingBlock(ctx, pendingBlk.Block); err != nil {
		log.WithError(err).Warn("could not publish un-confirmed block or cache it")
		return err
	}
	// Wait for final confirmation from orchestrator node
	if err := s.waitForConfirmationBlock(ctx, pendingBlk); err != nil {
		log.WithError(err).WithField("slot", pendingBlk.Block.Slot).Warn(
			"could not validate by orchestrator so discard the block")
		return err
	}
	return nil
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
		log.WithError(errInvalidRPCClient).Error("orchestrator rpc client is nil")
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
							"blockHash", fmt.Sprintf("%#x", data.BlockRootHash)).Debug(
							"got verified status from orchestrator")
						if err := s.pendingBlockCache.Delete(b.Block.Slot); err != nil {
							log.WithError(err).Error("couldn't delete the verified blocks from cache")
							return err
						}
						return nil
					case vanTypes.Pending:
						log.WithField("slot", data.Slot).WithField(
							"blockHash", fmt.Sprintf("%#x", data.BlockRootHash)).Debug(
							"got pending status from orchestrator")

						pendingBlockTryLimit = pendingBlockTryLimit - 1
						if pendingBlockTryLimit == 0 {
							log.WithField("slot", data.Slot).WithError(errPendingBlockTryLimitExceed).Error(
								"orchestrator sends pending status for this block so many times, deleting the invalid block from cache")

							if err := s.pendingBlockCache.Delete(data.Slot); err != nil {
								log.WithError(err).Error("couldn't delete the pending block from cache")
								return err
							}
							return errPendingBlockTryLimitExceed
						}
						continue
					case vanTypes.Invalid:
						log.WithField("slot", data.Slot).WithField(
							"blockHash", fmt.Sprintf("%#x", data.BlockRootHash)).Debug(
							"got invalid status from orchestrator, exiting goroutine")

						if err := s.pendingBlockCache.Delete(data.Slot); err != nil {
							log.WithError(err).Error("couldn't delete the invalid block from cache")
							return err
						}
						return errInvalidBlock
					case vanTypes.Skipped:
						log.WithError(errSkippedStatus).WithField("slot", data.Slot).WithField(
							"status", "skipped").Error(
							"got skipped status from orchestrator and discarding the block, exiting goroutine")
						if err := s.pendingBlockCache.Delete(data.Slot); err != nil {
							log.WithError(err).Error("couldn't delete the skipped block from cache")
							return err
						}
						return errSkippedStatus
					default:
						log.WithError(errUnknownStatus).WithField("slot", data.Slot).WithField(
							"status", "unknown").Error(
							"got unknown status from orchestrator and discarding the block, exiting goroutine")
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

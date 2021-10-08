package blockchain

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/interfaces"
	"github.com/prysmaticlabs/prysm/shared/event"
	vanTypes "github.com/prysmaticlabs/prysm/shared/params"
	"sort"
	"time"
)

var (
	// Getting confirmation status from orchestrator after each confirmationStatusFetchingInverval
	confirmationStatusFetchingInverval = 500 * time.Millisecond
	// maxPendingBlockTryLimit is the maximum limit for pending status of a block
	maxPendingBlockTryLimit          = 40
	errInvalidBlock                  = errors.New("invalid block found in orchestrator")
	errPendingBlockCtxIsDone         = errors.New("pending block confirmation context is done, reinitialize")
	errPendingBlockTryLimitExceed    = errors.New("maximum wait is exceeded and orchestrator can not verify the block")
	errUnknownStatus                 = errors.New("invalid status from orchestrator")
	errInvalidRPCClient              = errors.New("invalid orchestrator rpc client or no client initiated")
	errPendingQueueUnprocessed       = errors.New("pending queue is un-processed")
	errInvalidPandoraShardInfo       = errors.New("invalid pandora shard info")
	errInvalidPandoraShardInfoLength = errors.New("invalid pandora shard info length")
)

// PendingBlocksFetcher retrieves the cached un-confirmed beacon blocks from cache
type PendingBlocksFetcher interface {
	SortedUnConfirmedBlocksFromCache() ([]interfaces.BeaconBlock, error)
}

// BlockProposal interface use when validator calls GetBlock api for proposing new beancon block
type PendingQueueFetcher interface {
	CanPropose() error
	ActivateOrcVerification()
	DeactivateOrcVerification()
	OrcVerification() bool
}

// CanPropose
func (s *Service) CanPropose() error {
	blks, err := s.pendingBlockCache.PendingBlocks()
	if err != nil {
		return errors.Wrap(err, "Could not retrieve cached unconfirmed blocks from cache")
	}
	if len(blks) > 0 {
		log.WithField("unprocessedBlockLen", len(blks)).WithError(err).Error("Pending queue is not nil")
		return errPendingQueueUnprocessed
	}
	return nil
}

// UnConfirmedBlocksFromCache retrieves all the cached blocks from cache and send it back to event api
func (s *Service) SortedUnConfirmedBlocksFromCache() ([]interfaces.BeaconBlock, error) {
	blks, err := s.pendingBlockCache.PendingBlocks()
	if err != nil {
		return nil, errors.Wrap(err, "Could not retrieve cached unconfirmed blocks from cache")
	}
	sort.Slice(blks, func(i, j int) bool {
		return blks[i].Slot() < blks[j].Slot()
	})
	return blks, nil
}

// ActivateOrcVerification
func (s *Service) ActivateOrcVerification() {
	s.headLock.RLock()
	defer s.headLock.RUnlock()
	s.orcVerification = true
}

// DeactivateOrcVerification
func (s *Service) DeactivateOrcVerification() {
	s.headLock.RLock()
	defer s.headLock.RUnlock()
	s.orcVerification = false
}

// OrcVerification
func (s *Service) OrcVerification() bool {
	return s.orcVerification
}

// triggerEpochInfoPublisher publishes slot and state for publishing epoch info
func (s *Service) publishEpochInfo(
	slot types.Slot,
	proposerIndices []types.ValidatorIndex,
	pubKeys map[types.ValidatorIndex][48]byte,
) {
	// Send notification of the processed block to the state feed.
	s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.EpochInfo,
		Data: &statefeed.EpochInfoData{
			Slot:            slot,
			ProposerIndices: proposerIndices,
			PublicKeys:      pubKeys,
		},
	})
}

// publishBlock publishes downloaded blocks to orchestrator
func (s *Service) publishBlock(signedBlk interfaces.SignedBeaconBlock) {
	s.blockNotifier.BlockFeed().Send(&feed.Event{
		Type: blockfeed.UnConfirmedBlock,
		Data: &blockfeed.UnConfirmedBlockData{Block: signedBlk.Block()},
	})
}

// publishAndWaitForOrcConfirmation publish the block to orchestrator and store the block into pending queue cache
func (s *Service) waitForConfirmation(
	ctx context.Context,
	signedBlk interfaces.SignedBeaconBlock,
) error {
	// Storing pending block into pendingBlockCache
	if err := s.pendingBlockCache.AddPendingBlock(signedBlk.Block()); err != nil {
		return errors.Wrapf(err, "could not cache block of slot %d", signedBlk.Block().Slot())
	}
	// Wait for final confirmation from orchestrator node
	if err := s.waitForConfirmationBlock(ctx, signedBlk); err != nil {
		log.WithError(err).
			WithField("slot", signedBlk.Block().Slot()).
			Warn("could not validate by orchestrator so discard the block")
		return err
	}
	return nil
}

// processOrcConfirmation runs every certain interval and fetch confirmation from orchestrator periodically and
// publish the confirmation status to its subscriber methods. This loop will run in separate go routine when blockchain
// service starts.
func (s *Service) processOrcConfirmationRoutine() {
	ticker := time.NewTicker(confirmationStatusFetchingInverval)
	for {
		select {
		case <-ticker.C:
			if err := s.fetchConfirmations(s.ctx); err != nil {
				log.WithError(err).Error("Could not fetch confirmation from orchestrator")
			}
			continue
		case <-s.ctx.Done():
			log.WithField("function", "processOrcConfirmation").Debug("context is closed, exiting")
			ticker.Stop()
			return
		}
	}
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
func (s *Service) waitForConfirmationBlock(ctx context.Context, b interfaces.SignedBeaconBlock) error {
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
				if data.Slot == b.Block().Slot() {
					switch status := data.Status; status {
					case vanTypes.Verified:
						log.WithField("slot", data.Slot).WithField(
							"blockHash", fmt.Sprintf("%#x", data.BlockRootHash)).Debug(
							"got verified status from orchestrator")
						if err := s.pendingBlockCache.Delete(b.Block().Slot()); err != nil {
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
			Slot: blk.Slot(),
			Hash: blockRoot,
		})
	}

	sort.Slice(reqData, func(i, j int) bool {
		return reqData[i].Slot < reqData[j].Slot
	})

	return reqData, nil
}

func (s *Service) verifyPandoraShardInfo(signedBlk *ethpb.SignedBeaconBlock) error {
	if len(signedBlk.Block.Body.PandoraShard) == 0 {
		return errInvalidPandoraShardInfoLength
	}
	headBlk := s.headBlock()
	pandoraShards := headBlk.Block().Body().PandoraShards()
	if headBlk != nil && len(pandoraShards) > 0 {
		canonicalHash := common.BytesToHash(pandoraShards[0].Hash)
		canonicalBlkNum := pandoraShards[0].BlockNumber

		parentHash := common.BytesToHash(signedBlk.Block.Body.PandoraShard[0].ParentHash)
		blockNumber := signedBlk.Block.Body.PandoraShard[0].BlockNumber

		if parentHash != canonicalHash && blockNumber != canonicalBlkNum+1 {
			log.WithField("slot", signedBlk.Block.Slot).
				WithField("canonicalHash", canonicalHash).
				WithField("canonicalBlkNum", canonicalBlkNum).
				WithField("parentHash", parentHash).
				WithField("blockNumber", blockNumber).
				WithError(errInvalidPandoraShardInfo).
				Error("Failed to process block")
			return errInvalidPandoraShardInfo
		}
	}
	return nil
}

package blockchain

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/proto/interfaces"
	vanTypes "github.com/prysmaticlabs/prysm/shared/params"
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
	errInvalidPandoraShardInfo       = errors.New("invalid pandora shard info")
	errInvalidPandoraShardInfoLength = errors.New("invalid pandora shard info length")
	errInvalidRpcClientResLen        = errors.New("invalid length of orchestrator confirmation response")
	errInvalidConfirmationData       = errors.New("invalid orchestrator confirmation")
)

// ConfirmedData is the data which is sent after getting confirmation from orchestrator
type orcConfirmationData struct {
	slot          types.Slot
	blockRootHash [32]byte
	status        vanTypes.Status
}

// BlockProposal interface use when validator calls GetBlock api for proposing new beancon block
type PendingQueueFetcher interface {
	CanPropose() bool
	ActivateOrcVerification()
	DeactivateOrcVerification()
	OrcVerification() bool
}

// CanPropose
func (s *Service) CanPropose() bool {
	s.canProposeLock.RLock()
	defer s.canProposeLock.RUnlock()
	return s.canPropose
}

// ActivateOrcVerification
func (s *Service) ActivateOrcVerification() {
	s.orcVerificationLock.Lock()
	defer s.orcVerificationLock.Unlock()
	s.orcVerification = true
}

// DeactivateOrcVerification
func (s *Service) DeactivateOrcVerification() {
	s.orcVerificationLock.Lock()
	defer s.orcVerificationLock.Unlock()
	s.orcVerification = false
}

// OrcVerification
func (s *Service) OrcVerification() bool {
	s.orcVerificationLock.RLock()
	defer s.orcVerificationLock.RUnlock()
	return s.orcVerification
}

// SetLatestSentEpoch
func (s *Service) setLatestSentEpoch(epoch types.Epoch) {
	s.latestSentEpochLock.Lock()
	defer s.latestSentEpochLock.Unlock()
	s.latestSentEpoch = epoch
}

// GetLatestSentEpoch
func (s *Service) getLatestSentEpoch() types.Epoch {
	s.latestSentEpochLock.RLock()
	defer s.latestSentEpochLock.RUnlock()
	return s.latestSentEpoch
}

// deactivateBlockProposal
func (s *Service) deactivateBlockProposal() {
	s.canProposeLock.Lock()
	defer s.canProposeLock.Unlock()
	s.canPropose = false
}

// activateBlockProposal
func (s *Service) activateBlockProposal() {
	s.canProposeLock.Lock()
	defer s.canProposeLock.Unlock()
	s.canPropose = true
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

// fetchOrcConfirmations process confirmation for pending blocks
// -> After getting confirmation for a list of pending slots, it iterates through the list
// -> If any slot gets invalid status then stop the iteration and start again from that slot
// -> If any slot gets verified status then, publish the slots and block hashes to the blockchain service
//    who actually waiting for confirmed blocks
// -> If any slot gets un
func (s *Service) fetchConfirmations(signedBlk interfaces.SignedBeaconBlock) (*orcConfirmationData, error) {
	blockRoot, err := signedBlk.Block().HashTreeRoot()
	if err != nil {
		return nil, err
	}
	reqData := make([]*vanTypes.ConfirmationReqData, 0, 1) // only one block needs confirmation at a time
	reqData = append(reqData, &vanTypes.ConfirmationReqData{
		Slot: signedBlk.Block().Slot(),
		Hash: blockRoot,
	})
	if s.orcRPCClient == nil {
		log.WithError(errInvalidRPCClient).Error("orchestrator rpc client is nil")
		return nil, errInvalidRPCClient
	}
	resData, err := s.orcRPCClient.ConfirmVanBlockHashes(s.ctx, reqData)
	if err != nil {
		return nil, err
	}
	if len(resData) < 1 {
		return nil, errInvalidRpcClientResLen
	}
	return &orcConfirmationData{
		slot:          resData[0].Slot,
		blockRootHash: resData[0].Hash,
		status:        resData[0].Status,
	}, nil
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
func (s *Service) waitForConfirmation(b interfaces.SignedBeaconBlock) error {
	// first de-activated the block proposal process
	s.deactivateBlockProposal()
	defer s.activateBlockProposal()

	pendingBlockTryLimit := maxPendingBlockTryLimit
	ticker := time.NewTicker(confirmationStatusFetchingInverval)
	for {
		select {
		case <-ticker.C:
			// fetching confirmation for this block from orchestrator
			responseData, err := s.fetchConfirmations(b)
			if err != nil {
				return errors.Wrap(err, "could not fetch confirmation from orchestrator")
			}
			// Checks slot number with incoming confirmation data slot
			if responseData.slot != b.Block().Slot() {
				return errors.Wrap(errInvalidConfirmationData, "block slot mismatched with response data")
			}
			commonLog := log.WithField("slot", responseData.slot).WithField("blockHash", fmt.Sprintf("%#x", responseData.blockRootHash))
			switch status := responseData.status; status {
			case vanTypes.Verified:
				commonLog.Debug("got verified status from orchestrator")
				return nil
			case vanTypes.Pending:
				commonLog.Debug("got pending status from orchestrator")
				pendingBlockTryLimit = pendingBlockTryLimit - 1
				if pendingBlockTryLimit == 0 {
					log.WithField("slot", responseData.slot).WithError(errPendingBlockTryLimitExceed).Error(
						"orchestrator sends pending status for this block so many times, discard this invalid block")
					return errPendingBlockTryLimitExceed
				}
				continue
			case vanTypes.Invalid:
				commonLog.Debug("got invalid status from orchestrator, exiting goroutine")
				return errInvalidBlock
			default:
				log.WithError(errUnknownStatus).WithField("slot", responseData.slot).WithField("status", "unknown").Error(
					"got unknown status from orchestrator and discarding the block, exiting goroutine")
				return errUnknownStatus
			}
		case <-s.ctx.Done():
			log.WithField("function", "waitForConfirmationBlock").Debug("context is closed, exiting")
			return errPendingBlockCtxIsDone
		}
	}
}

// verifyPandoraShardInfo
func (s *Service) verifyPandoraShardInfo(signedBlk interfaces.SignedBeaconBlock) error {
	// For slot #1, we don't have shard info for previous block so short circuit here
	if signedBlk.Block().Slot() == 1 {
		return nil
	}
	// Checking length of current block's pandora shard info
	curPandoraShards := signedBlk.Block().Body().PandoraShards()
	if len(curPandoraShards) == 0 {
		return errInvalidPandoraShardInfoLength
	}
	// Checking current block pandora shard's parent with canonical head's pandora shard's header hash
	canonicalHeadBlock := s.head.block
	if canonicalHeadBlock != nil && len(canonicalHeadBlock.Block().Body().PandoraShards()) > 0 {
		parentPandoraShards := canonicalHeadBlock.Block().Body().PandoraShards()
		canonicalShardingHash := common.BytesToHash(parentPandoraShards[0].Hash)
		canonicalShardingBlkNum := parentPandoraShards[0].BlockNumber

		curShardingParentHash := common.BytesToHash(curPandoraShards[0].ParentHash)
		curShardingBlockNumber := curPandoraShards[0].BlockNumber
		commonLog := log.WithField("slot", signedBlk.Block().Slot()).WithField("canonicalShardingHash", canonicalShardingHash).
			WithField("canonicalShardingBlkNum", canonicalShardingBlkNum).WithField("curShardingParentHash", curShardingParentHash).
			WithField("curShardingBlockNumber", curShardingBlockNumber)

		if curShardingParentHash != canonicalShardingHash && curShardingBlockNumber != canonicalShardingBlkNum+1 {
			commonLog.WithError(errInvalidPandoraShardInfo).Error("Failed to verify pandora sharding info")
			return errInvalidPandoraShardInfo
		}
		commonLog.Debug("Successfully verified pandora sharding info")
	}
	return nil
}

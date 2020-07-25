package powchain

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	protodb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var (
	depositEventSignature = hashutil.HashKeccak256([]byte("DepositEvent(bytes,bytes,bytes,bytes,bytes)"))
)

const eth1LookBackPeriod = 100
const eth1DataSavingInterval = 100
const eth1HeaderReqLimit = 1000
const depositlogRequestLimit = 10000

// Eth2GenesisPowchainInfo retrieves the genesis time and eth1 block number of the beacon chain
// from the deposit contract.
func (s *Service) Eth2GenesisPowchainInfo() (uint64, *big.Int) {
	return s.chainStartData.GenesisTime, big.NewInt(int64(s.chainStartData.GenesisBlock))
}

// ProcessETH1Block processes the logs from the provided eth1Block.
func (s *Service) ProcessETH1Block(ctx context.Context, blkNum *big.Int) error {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			s.depositContractAddress,
		},
		FromBlock: blkNum,
		ToBlock:   blkNum,
	}
	logs, err := s.httpLogger.FilterLogs(ctx, query)
	if err != nil {
		return err
	}
	for _, log := range logs {
		// ignore logs that are not of the required block number
		if log.BlockNumber != blkNum.Uint64() {
			continue
		}
		if err := s.ProcessLog(ctx, log); err != nil {
			return errors.Wrap(err, "could not process log")
		}
	}
	if !s.chainStartData.Chainstarted {
		if err := s.checkBlockNumberForChainStart(ctx, blkNum); err != nil {
			return err
		}
	}
	return nil
}

// ProcessLog is the main method which handles the processing of all
// logs from the deposit contract on the ETH1.0 chain.
func (s *Service) ProcessLog(ctx context.Context, depositLog gethTypes.Log) error {
	s.processingLock.RLock()
	defer s.processingLock.RUnlock()
	// Process logs according to their event signature.
	if depositLog.Topics[0] == depositEventSignature {
		if err := s.ProcessDepositLog(ctx, depositLog); err != nil {
			return errors.Wrap(err, "Could not process deposit log")
		}
		if s.lastReceivedMerkleIndex%eth1DataSavingInterval == 0 {
			eth1Data := &protodb.ETH1ChainData{
				CurrentEth1Data:   s.latestEth1Data,
				ChainstartData:    s.chainStartData,
				BeaconState:       s.preGenesisState.InnerStateUnsafe(), // I promise not to mutate it!
				Trie:              s.depositTrie.ToProto(),
				DepositContainers: s.depositCache.AllDepositContainers(ctx),
			}
			return s.beaconDB.SavePowchainData(ctx, eth1Data)
		}
		return nil
	}
	log.WithField("signature", fmt.Sprintf("%#x", depositLog.Topics[0])).Debug("Not a valid event signature")
	return nil
}

// ProcessDepositLog processes the log which had been received from
// the ETH1.0 chain by trying to ascertain which participant deposited
// in the contract.
func (s *Service) ProcessDepositLog(ctx context.Context, depositLog gethTypes.Log) error {
	pubkey, withdrawalCredentials, amount, signature, merkleTreeIndex, err := contracts.UnpackDepositLogData(depositLog.Data)
	if err != nil {
		return errors.Wrap(err, "Could not unpack log")
	}
	// If we have already seen this Merkle index, skip processing the log.
	// This can happen sometimes when we receive the same log twice from the
	// ETH1.0 network, and prevents us from updating our trie
	// with the same log twice, causing an inconsistent state root.
	index := binary.LittleEndian.Uint64(merkleTreeIndex)
	if int64(index) <= s.lastReceivedMerkleIndex {
		return nil
	}

	if int64(index) != s.lastReceivedMerkleIndex+1 {
		missedDepositLogsCount.Inc()
		if s.requestingOldLogs {
			return errors.New("received incorrect merkle index")
		}
		if err := s.requestMissingLogs(ctx, depositLog.BlockNumber, int64(index-1)); err != nil {
			return errors.Wrap(err, "could not get correct merkle index")
		}

	}
	s.lastReceivedMerkleIndex = int64(index)

	// We then decode the deposit input in order to create a deposit object
	// we can store in our persistent DB.
	depositData := &ethpb.Deposit_Data{
		Amount:                bytesutil.FromBytes8(amount),
		PublicKey:             pubkey,
		Signature:             signature,
		WithdrawalCredentials: withdrawalCredentials,
	}

	depositHash, err := depositData.HashTreeRoot()
	if err != nil {
		return errors.Wrap(err, "Unable to determine hashed value of deposit")
	}

	s.depositTrie.Insert(depositHash[:], int(index))

	proof, err := s.depositTrie.MerkleProof(int(index))
	if err != nil {
		return errors.Wrap(err, "Unable to generate merkle proof for deposit")
	}

	deposit := &ethpb.Deposit{
		Data:  depositData,
		Proof: proof,
	}

	// Make sure duplicates are rejected pre-chainstart.
	if !s.chainStartData.Chainstarted {
		var pubkey = fmt.Sprintf("%#x", depositData.PublicKey)
		if s.depositCache.PubkeyInChainstart(ctx, pubkey) {
			log.WithField("publicKey", pubkey).Debug("Pubkey has already been submitted for chainstart")
		} else {
			s.depositCache.MarkPubkeyForChainstart(ctx, pubkey)
		}
	}

	// We always store all historical deposits in the DB.
	s.depositCache.InsertDeposit(ctx, deposit, depositLog.BlockNumber, int64(index), s.depositTrie.Root())
	validData := true
	if !s.chainStartData.Chainstarted {
		s.chainStartData.ChainstartDeposits = append(s.chainStartData.ChainstartDeposits, deposit)
		root := s.depositTrie.Root()
		eth1Data := &ethpb.Eth1Data{
			DepositRoot:  root[:],
			DepositCount: uint64(len(s.chainStartData.ChainstartDeposits)),
		}
		if err := s.processDeposit(eth1Data, deposit); err != nil {
			log.Errorf("Invalid deposit processed: %v", err)
			validData = false
		}
	} else {
		s.depositCache.InsertPendingDeposit(ctx, deposit, depositLog.BlockNumber, int64(index), s.depositTrie.Root())
	}
	if validData {
		log.WithFields(logrus.Fields{
			"eth1Block":       depositLog.BlockNumber,
			"publicKey":       fmt.Sprintf("%#x", depositData.PublicKey),
			"merkleTreeIndex": index,
		}).Debug("Deposit registered from deposit contract")
		validDepositsCount.Inc()
		// Notify users what is going on, from time to time.
		if !s.chainStartData.Chainstarted {
			deposits := len(s.chainStartData.ChainstartDeposits)
			if deposits%512 == 0 {
				valCount, err := helpers.ActiveValidatorCount(s.preGenesisState, 0)
				if err != nil {
					log.WithError(err).Error("Could not determine active validator count from pre genesis state")
				}
				log.WithFields(logrus.Fields{
					"deposits":          deposits,
					"genesisValidators": valCount,
				}).Info("Processing deposits from Ethereum 1 chain")
			}
		}
	} else {
		log.WithFields(logrus.Fields{
			"eth1Block":       depositLog.BlockHash.Hex(),
			"eth1Tx":          depositLog.TxHash.Hex(),
			"merkleTreeIndex": index,
		}).Info("Invalid deposit registered in deposit contract")
	}
	return nil
}

// ProcessChainStart processes the log which had been received from
// the ETH1.0 chain by trying to determine when to start the beacon chain.
func (s *Service) ProcessChainStart(genesisTime uint64, eth1BlockHash [32]byte, blockNumber *big.Int) {
	s.chainStartData.Chainstarted = true
	s.chainStartData.GenesisBlock = blockNumber.Uint64()

	chainStartTime := time.Unix(int64(genesisTime), 0)

	for i := range s.chainStartData.ChainstartDeposits {
		proof, err := s.depositTrie.MerkleProof(i)
		if err != nil {
			log.Errorf("Unable to generate deposit proof %v", err)
		}
		s.chainStartData.ChainstartDeposits[i].Proof = proof
	}

	root := s.depositTrie.Root()
	s.chainStartData.Eth1Data = &ethpb.Eth1Data{
		DepositCount: uint64(len(s.chainStartData.ChainstartDeposits)),
		DepositRoot:  root[:],
		BlockHash:    eth1BlockHash[:],
	}

	log.WithFields(logrus.Fields{
		"ChainStartTime": chainStartTime,
	}).Info("Minimum number of validators reached for beacon-chain to start")
	s.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.ChainStarted,
		Data: &statefeed.ChainStartedData{
			StartTime: chainStartTime,
		},
	})
}

func (s *Service) createGenesisTime(timeStamp uint64) uint64 {
	// adds in the genesis delay to the eth1 block time
	// on which it was triggered.
	return timeStamp + cmd.Get().CustomGenesisDelay
}

// processPastLogs processes all the past logs from the deposit contract and
// updates the deposit trie with the data from each individual log.
func (s *Service) processPastLogs(ctx context.Context) error {
	currentBlockNum := s.latestEth1Data.LastRequestedBlock
	deploymentBlock := int64(params.BeaconNetworkConfig().ContractDeploymentBlock)
	if uint64(deploymentBlock) > currentBlockNum {
		currentBlockNum = uint64(deploymentBlock)
	}
	// To store all blocks.
	headersMap := make(map[uint64]*gethTypes.Header)
	rawLogCount, err := s.depositContractCaller.GetDepositCount(&bind.CallOpts{})
	if err != nil {
		return err
	}
	logCount := binary.LittleEndian.Uint64(rawLogCount)

	// Batch request the desired headers and store them in a
	// map for quick access.
	requestHeaders := func(startBlk uint64, endBlk uint64) error {
		headers, err := s.batchRequestHeaders(startBlk, endBlk)
		if err != nil {
			return err
		}
		for _, h := range headers {
			if h != nil && h.Number != nil {
				headersMap[h.Number.Uint64()] = h
			}
		}
		return nil
	}
	latestFollowHeight, err := s.followBlockHeight(ctx)
	if err != nil {
		return err
	}
	for currentBlockNum < latestFollowHeight {
		// stop requesting, if we have all the logs
		if logCount == uint64(s.lastReceivedMerkleIndex+1) {
			break
		}
		start := currentBlockNum
		end := currentBlockNum + eth1HeaderReqLimit
		if end > latestFollowHeight {
			end = latestFollowHeight
		}
		query := ethereum.FilterQuery{
			Addresses: []common.Address{
				s.depositContractAddress,
			},
			FromBlock: big.NewInt(int64(start)),
			ToBlock:   big.NewInt(int64(end)),
		}
		remainingLogs := logCount - uint64(s.lastReceivedMerkleIndex+1)
		// only change the end block if the remaining logs are below the required log limit.
		if remainingLogs < depositlogRequestLimit && end >= latestFollowHeight {
			query.ToBlock = big.NewInt(int64(latestFollowHeight))
			end = latestFollowHeight
		}
		logs, err := s.httpLogger.FilterLogs(ctx, query)
		if err != nil {
			return err
		}
		if !s.chainStartData.Chainstarted {
			if err := requestHeaders(start, end); err != nil {
				return err
			}
		}

		for _, log := range logs {
			if log.BlockNumber > currentBlockNum {
				if err := s.checkHeaderRange(currentBlockNum, log.BlockNumber-1, headersMap, requestHeaders); err != nil {
					return err
				}
				// set new block number after checking for chainstart for previous block.
				s.latestEth1Data.LastRequestedBlock = currentBlockNum
				currentBlockNum = log.BlockNumber
			}
			if err := s.ProcessLog(ctx, log); err != nil {
				return err
			}
		}
		if err := s.checkHeaderRange(currentBlockNum, end, headersMap, requestHeaders); err != nil {
			return err
		}
		currentBlockNum = end
	}

	s.latestEth1Data.LastRequestedBlock = currentBlockNum
	currentState, err := s.beaconDB.HeadState(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head state")
	}

	if currentState != nil && currentState.Eth1DepositIndex() > 0 {
		s.depositCache.PrunePendingDeposits(ctx, int64(currentState.Eth1DepositIndex()))
	}

	return nil
}

// requestBatchedLogs requests and processes all the logs from the period
// last polled to now.
func (s *Service) requestBatchedLogs(ctx context.Context) error {
	// We request for the nth block behind the current head, in order to have
	// stabilized logs when we retrieve it from the 1.0 chain.

	requestedBlock, err := s.followBlockHeight(ctx)
	if err != nil {
		return err
	}
	for i := s.latestEth1Data.LastRequestedBlock + 1; i <= requestedBlock; i++ {
		err := s.ProcessETH1Block(ctx, big.NewInt(int64(i)))
		if err != nil {
			return err
		}
		s.latestEth1Data.LastRequestedBlock = i
	}

	return nil
}

// requestMissingLogs requests any logs that were missed by requesting from previous blocks
// until the current block(exclusive).
func (s *Service) requestMissingLogs(ctx context.Context, blkNumber uint64, wantedIndex int64) error {
	// Prevent this method from being called recursively
	s.requestingOldLogs = true
	defer func() {
		s.requestingOldLogs = false
	}()
	// We request from the last requested block till the current block(exclusive)
	beforeCurrentBlk := big.NewInt(int64(blkNumber) - 1)
	startBlock := s.latestEth1Data.LastRequestedBlock + 1
	for {
		err := s.processBlksInRange(ctx, startBlock, beforeCurrentBlk.Uint64())
		if err != nil {
			return err
		}

		if s.lastReceivedMerkleIndex == wantedIndex {
			break
		}

		// If the required logs still do not exist after the lookback period, then we return an error.
		if startBlock < s.latestEth1Data.LastRequestedBlock-eth1LookBackPeriod {
			return fmt.Errorf(
				"latest index observed is not accurate, wanted %d, but received  %d",
				wantedIndex,
				s.lastReceivedMerkleIndex,
			)
		}
		startBlock--
	}
	return nil
}

func (s *Service) processBlksInRange(ctx context.Context, startBlk uint64, endBlk uint64) error {
	for i := startBlk; i <= endBlk; i++ {
		err := s.ProcessETH1Block(ctx, big.NewInt(int64(i)))
		if err != nil {
			return err
		}
	}
	return nil
}

// checkBlockNumberForChainStart checks the given block number for if chainstart has occurred.
func (s *Service) checkBlockNumberForChainStart(ctx context.Context, blkNum *big.Int) error {
	hash, err := s.BlockHashByHeight(ctx, blkNum)
	if err != nil {
		return errors.Wrap(err, "could not get eth1 block hash")
	}
	if hash == [32]byte{} {
		return errors.Wrap(err, "got empty block hash")
	}

	timeStamp, err := s.BlockTimeByHeight(ctx, blkNum)
	if err != nil {
		return errors.Wrap(err, "could not get block timestamp")
	}
	s.checkForChainstart(hash, blkNum, timeStamp)
	return nil
}

func (s *Service) checkHeaderForChainstart(header *gethTypes.Header) {
	s.checkForChainstart(header.Hash(), header.Number, header.Time)
}

func (s *Service) checkHeaderRange(start uint64, end uint64,
	headersMap map[uint64]*gethTypes.Header,
	requestHeaders func(uint64, uint64) error) error {
	for i := start; i <= end; i++ {
		if !s.chainStartData.Chainstarted {
			h, ok := headersMap[i]
			if !ok {
				if err := requestHeaders(i, end); err != nil {
					return err
				}
				// Retry this block.
				i--
				continue
			}
			s.checkHeaderForChainstart(h)
		}
	}
	return nil
}

func (s *Service) checkForChainstart(blockHash [32]byte, blockNumber *big.Int, blockTime uint64) {
	valCount, err := helpers.ActiveValidatorCount(s.preGenesisState, 0)
	if err != nil {
		log.WithError(err).Error("Could not determine active validator count from pref genesis state")
	}
	triggered := state.IsValidGenesisState(valCount, s.createGenesisTime(blockTime))
	if triggered {
		s.chainStartData.GenesisTime = s.createGenesisTime(blockTime)
		s.ProcessChainStart(s.chainStartData.GenesisTime, blockHash, blockNumber)
	}
}

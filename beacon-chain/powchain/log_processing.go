package powchain

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

var (
	depositEventSignature = []byte("DepositEvent(bytes,bytes,bytes,bytes,bytes)")
)

// Eth2GenesisPowchainInfo retrieves the genesis time and eth1 block number of the beacon chain
// from the deposit contract.
func (s *Service) Eth2GenesisPowchainInfo() (uint64, *big.Int) {
	return s.eth2GenesisTime, s.chainStartBlockNumber
}

// ProcessLog is the main method which handles the processing of all
// logs from the deposit contract on the ETH1.0 chain.
func (s *Service) ProcessLog(ctx context.Context, depositLog gethTypes.Log) error {
	s.processingLock.RLock()
	defer s.processingLock.RUnlock()
	// Process logs according to their event signature.
	if depositLog.Topics[0] == hashutil.HashKeccak256(depositEventSignature) {
		if err := s.ProcessDepositLog(ctx, depositLog); err != nil {
			return errors.Wrap(err, "Could not process deposit log")
		}
		if !s.chainStarted {
			if depositLog.BlockHash == [32]byte{} {
				return errors.New("got empty blockhash from powchain service")
			}
			blk, err := s.blockFetcher.BlockByHash(ctx, depositLog.BlockHash)
			if err != nil {
				return errors.Wrap(err, "could not get eth1 block")
			}
			if blk == nil {
				return errors.Wrap(err, "got empty block from powchain service")
			}
			timeStamp := blk.Time()
			triggered := state.IsValidGenesisState(s.activeValidatorCount, timeStamp)
			if triggered {
				s.setGenesisTime(timeStamp)
				s.ProcessChainStart(uint64(s.eth2GenesisTime), depositLog.BlockHash, blk.Number())
			}
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
		if err := s.requestMissingLogs(ctx, depositLog.BlockNumber, int64(index-1)); err != nil {
			return errors.Wrap(err, "Could not get correct merkle index")
		}
	}
	s.lastReceivedMerkleIndex = int64(index)

	// We then decode the deposit input in order to create a deposit object
	// we can store in our persistent DB.
	validData := true
	depositData := &ethpb.Deposit_Data{
		Amount:                bytesutil.FromBytes8(amount),
		PublicKey:             pubkey,
		Signature:             signature,
		WithdrawalCredentials: withdrawalCredentials,
	}

	depositHash, err := ssz.HashTreeRoot(depositData)
	if err != nil {
		return errors.Wrap(err, "Unable to determine hashed value of deposit")
	}

	if err := s.depositTrie.InsertIntoTrie(depositHash[:], int(index)); err != nil {
		return errors.Wrap(err, "Unable to insert deposit into trie")
	}

	proof, err := s.depositTrie.MerkleProof(int(index))
	if err != nil {
		return errors.Wrap(err, "Unable to generate merkle proof for deposit")
	}

	deposit := &ethpb.Deposit{
		Data:  depositData,
		Proof: proof,
	}

	// Make sure duplicates are rejected pre-chainstart.
	if !s.chainStarted && validData {
		var pubkey = fmt.Sprintf("#%x", depositData.PublicKey)
		if s.depositCache.PubkeyInChainstart(ctx, pubkey) {
			log.Warnf("Pubkey %#x has already been submitted for chainstart", pubkey)
		} else {
			s.depositCache.MarkPubkeyForChainstart(ctx, pubkey)
		}

	}

	// We always store all historical deposits in the DB.
	s.depositCache.InsertDeposit(ctx, deposit, big.NewInt(int64(depositLog.BlockNumber)), int(index), s.depositTrie.Root())

	if !s.chainStarted {
		s.chainStartDeposits = append(s.chainStartDeposits, deposit)
		root := s.depositTrie.Root()
		eth1Data := &ethpb.Eth1Data{
			DepositRoot:  root[:],
			DepositCount: uint64(len(s.chainStartDeposits)),
		}
		if err := s.processDeposit(eth1Data, deposit); err != nil {
			log.Errorf("Invalid deposit processed: %v", err)
			validData = false
		}
	} else {
		s.depositCache.InsertPendingDeposit(ctx, deposit, big.NewInt(int64(depositLog.BlockNumber)), int(index), s.depositTrie.Root())
	}
	if validData {
		log.WithFields(logrus.Fields{
			"publicKey":       fmt.Sprintf("%#x", depositData.PublicKey),
			"merkleTreeIndex": index,
		}).Debug("Deposit registered from deposit contract")
		validDepositsCount.Inc()
	} else {
		log.WithFields(logrus.Fields{
			"merkleTreeIndex": index,
		}).Info("Invalid deposit registered in deposit contract")
	}
	return nil
}

// ProcessChainStart processes the log which had been received from
// the ETH1.0 chain by trying to determine when to start the beacon chain.
func (s *Service) ProcessChainStart(genesisTime uint64, eth1BlockHash [32]byte, blockNumber *big.Int) {
	s.chainStarted = true
	s.chainStartBlockNumber = blockNumber

	chainStartTime := time.Unix(int64(genesisTime), 0)
	depHashes, err := s.ChainStartDepositHashes()
	if err != nil {
		log.Errorf("Generating chainstart deposit hashes failed: %v", err)
		return
	}

	// We then update the in-memory deposit trie from the chain start
	// deposits at this point, as this trie will be later needed for
	// incoming, post-chain start deposits.
	sparseMerkleTrie, err := trieutil.GenerateTrieFromItems(
		depHashes,
		int(params.BeaconConfig().DepositContractTreeDepth),
	)
	if err != nil {
		log.Fatalf("Unable to generate deposit trie from ChainStart deposits: %v", err)
	}

	for i := range s.chainStartDeposits {
		proof, err := sparseMerkleTrie.MerkleProof(i)
		if err != nil {
			log.Errorf("Unable to generate deposit proof %v", err)
		}
		s.chainStartDeposits[i].Proof = proof
	}

	s.depositTrie = sparseMerkleTrie
	root := sparseMerkleTrie.Root()
	s.chainStartETH1Data = &ethpb.Eth1Data{
		DepositCount: uint64(len(s.chainStartDeposits)),
		DepositRoot:  root[:],
		BlockHash:    eth1BlockHash[:],
	}

	log.WithFields(logrus.Fields{
		"ChainStartTime": chainStartTime,
	}).Info("Minimum number of validators reached for beacon-chain to start")
	s.chainStartFeed.Send(chainStartTime)
}

func (s *Service) setGenesisTime(timeStamp uint64) {
	if featureconfig.FeatureConfig().NoGenesisDelay {
		s.eth2GenesisTime = uint64(time.Unix(int64(timeStamp), 0).Add(30 * time.Second).Unix())
	} else {
		timeStampRdDown := timeStamp - timeStamp%params.BeaconConfig().SecondsPerDay
		// genesisTime will be set to the first second of the day, two days after it was triggered.
		s.eth2GenesisTime = timeStampRdDown + 2*params.BeaconConfig().SecondsPerDay
	}
}

// processPastLogs processes all the past logs from the deposit contract and
// updates the deposit trie with the data from each individual log.
func (s *Service) processPastLogs(ctx context.Context) error {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			s.depositContractAddress,
		},
	}

	logs, err := s.httpLogger.FilterLogs(ctx, query)
	if err != nil {
		return err
	}

	for _, log := range logs {
		if err := s.ProcessLog(ctx, log); err != nil {
			return errors.Wrap(err, "could not process log")
		}
	}
	s.lastRequestedBlock.Set(s.blockHeight)

	currentState, err := s.beaconDB.HeadState(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head state")
	}

	if currentState != nil && currentState.Eth1DepositIndex > 0 {
		s.depositCache.PrunePendingDeposits(ctx, int(currentState.Eth1DepositIndex))
	}

	return nil
}

// requestBatchedLogs requests and processes all the logs from the period
// last polled to now.
func (s *Service) requestBatchedLogs(ctx context.Context) error {
	// We request for the nth block behind the current head, in order to have
	// stabilized logs when we retrieve it from the 1.0 chain.
	requestedBlock := big.NewInt(0).Sub(s.blockHeight, big.NewInt(params.BeaconConfig().LogBlockDelay))
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			s.depositContractAddress,
		},
		FromBlock: s.lastRequestedBlock.Add(s.lastRequestedBlock, big.NewInt(1)),
		ToBlock:   requestedBlock,
	}
	logs, err := s.httpLogger.FilterLogs(ctx, query)
	if err != nil {
		return err
	}

	// Only process log slices which are larger than zero.
	if len(logs) > 0 {
		for _, log := range logs {
			if err := s.ProcessLog(ctx, log); err != nil {
				return errors.Wrap(err, "could not process log")
			}
		}
	}

	s.lastRequestedBlock.Set(requestedBlock)
	return nil
}

// requestMissingLogs requests any logs that were missed by requesting from previous blocks
// until the current block(exclusive).
func (s *Service) requestMissingLogs(ctx context.Context, blkNumber uint64, wantedIndex int64) error {
	// We request from the last requested block till the current block(exclusive)
	beforeCurrentBlk := big.NewInt(int64(blkNumber) - 1)
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			s.depositContractAddress,
		},
		FromBlock: big.NewInt(0).Add(s.lastRequestedBlock, big.NewInt(1)),
		ToBlock:   beforeCurrentBlk,
	}
	logs, err := s.httpLogger.FilterLogs(ctx, query)
	if err != nil {
		return err
	}

	// Only process log slices which are larger than zero.
	if len(logs) > 0 {
		for _, log := range logs {
			if err := s.ProcessLog(ctx, log); err != nil {
				return errors.Wrap(err, "could not process log")
			}
		}
	}

	if s.lastReceivedMerkleIndex != wantedIndex {
		return fmt.Errorf("despite requesting missing logs, latest index observed is not accurate. "+
			"Wanted %d but got %d", wantedIndex, s.lastReceivedMerkleIndex)
	}
	return nil
}

// ChainStartDepositHashes returns the hashes of all the chainstart deposits
// stored in memory.
func (s *Service) ChainStartDepositHashes() ([][]byte, error) {
	hashes := make([][]byte, len(s.chainStartDeposits))
	for i, dep := range s.chainStartDeposits {
		hash, err := ssz.HashTreeRoot(dep.Data)
		if err != nil {
			return nil, err
		}
		hashes[i] = hash[:]
	}
	return hashes, nil
}

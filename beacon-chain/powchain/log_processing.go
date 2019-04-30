package powchain

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

var (
	depositEventSignature    = []byte("Deposit(bytes32,bytes,bytes,bytes32[32])")
	chainStartEventSignature = []byte("ChainStart(bytes32,bytes)")
)

// HasChainStartLogOccurred queries all logs in the deposit contract to verify
// if ChainStart has occurred. If so, it returns true alongside the ChainStart timestamp.
func (w *Web3Service) HasChainStartLogOccurred() (bool, uint64, error) {
	genesisTime, err := w.depositContractCaller.GenesisTime(&bind.CallOpts{})
	if err != nil {
		return false, 0, fmt.Errorf("could not query contract to verify chain started: %v", err)
	}
	// If chain has not yet started, the result will be an empty byte slice.
	if bytes.Equal(genesisTime, []byte{}) {
		return false, 0, nil
	}
	timestamp := binary.LittleEndian.Uint64(genesisTime)
	return true, timestamp, nil
}

// ProcessLog is the main method which handles the processing of all
// logs from the deposit contract on the ETH1.0 chain.
func (w *Web3Service) ProcessLog(depositLog gethTypes.Log) {
	// Process logs according to their event signature.
	if depositLog.Topics[0] == hashutil.Hash(depositEventSignature) {
		w.ProcessDepositLog(depositLog)
		return
	}
	if depositLog.Topics[0] == hashutil.Hash(chainStartEventSignature) && !w.chainStarted {
		w.ProcessChainStartLog(depositLog)
		return
	}
	log.WithField("signature", fmt.Sprintf("%#x", depositLog.Topics[0])).Debug("Not a valid signature")
}

// ProcessDepositLog processes the log which had been received from
// the ETH1.0 chain by trying to ascertain which participant deposited
// in the contract.
func (w *Web3Service) ProcessDepositLog(depositLog gethTypes.Log) {
	_, depositData, merkleTreeIndex, _, err := contracts.UnpackDepositLogData(depositLog.Data)
	if err != nil {
		log.Errorf("Could not unpack log %v", err)
		return
	}
	// If we have already seen this Merkle index, skip processing the log.
	// This can happen sometimes when we receive the same log twice from the
	// ETH1.0 network, and prevents us from updating our trie
	// with the same log twice, causing an inconsistent state root.
	index := binary.LittleEndian.Uint64(merkleTreeIndex)
	if int64(index) <= w.lastReceivedMerkleIndex {
		return
	}
	w.lastReceivedMerkleIndex = int64(index)

	// We then decode the deposit input in order to create a deposit object
	// we can store in our persistent DB.
	validData := true
	depositInput, err := helpers.DecodeDepositInput(depositData)
	if err != nil {
		log.Debugf("Could not decode deposit input %v", err)
		validData = false
	}

	deposit := &pb.Deposit{
		DepositData:     depositData,
		MerkleTreeIndex: index,
	}

	// Make sure duplicates are rejected pre-chainstart.
	if !w.chainStarted && validData {
		var pubkey = fmt.Sprintf("#%x", depositInput.Pubkey)
		if w.beaconDB.PubkeyInChainstart(w.ctx, pubkey) {
			log.Warnf("Pubkey %#x has already been submitted for chainstart", pubkey)
		} else {
			w.beaconDB.MarkPubkeyForChainstart(w.ctx, pubkey)
		}
	}

	// We always store all historical deposits in the DB.
	w.beaconDB.InsertDeposit(w.ctx, deposit, big.NewInt(int64(depositLog.BlockNumber)))

	if !w.chainStarted {
		w.chainStartDeposits = append(w.chainStartDeposits, depositData)
	} else {
		w.beaconDB.InsertPendingDeposit(w.ctx, deposit, big.NewInt(int64(depositLog.BlockNumber)))
	}
	if validData {
		log.WithFields(logrus.Fields{
			"publicKey":       fmt.Sprintf("%#x", depositInput.Pubkey),
			"merkleTreeIndex": index,
		}).Debug("Deposit registered from deposit contract")
		validDepositsCount.Inc()
	} else {
		log.WithFields(logrus.Fields{
			"merkleTreeIndex": index,
		}).Debug("Invalid deposit registered in deposit contract")
	}
}

// ProcessChainStartLog processes the log which had been received from
// the ETH1.0 chain by trying to determine when to start the beacon chain.
func (w *Web3Service) ProcessChainStartLog(depositLog gethTypes.Log) {
	chainStartCount.Inc()
	chainStartDepositRoot, timestampData, err := contracts.UnpackChainStartLogData(depositLog.Data)
	if err != nil {
		log.Errorf("Unable to unpack ChainStart log data %v", err)
		return
	}

	w.chainStartETH1Data = &pb.Eth1Data{
		BlockHash32:       depositLog.BlockHash[:],
		DepositRootHash32: chainStartDepositRoot[:],
	}

	timestamp := binary.LittleEndian.Uint64(timestampData)
	w.chainStarted = true
	w.depositRoot = chainStartDepositRoot[:]
	chainStartTime := time.Unix(int64(timestamp), 0)

	// We then update the in-memory deposit trie from the chain start
	// deposits at this point, as this trie will be later needed for
	// incoming, post-chain start deposits.
	sparseMerkleTrie, err := trieutil.GenerateTrieFromItems(
		w.chainStartDeposits,
		int(params.BeaconConfig().DepositContractTreeDepth),
	)
	if err != nil {
		log.Fatalf("Unable to generate deposit trie from ChainStart deposits: %v", err)
	}
	w.depositTrie = sparseMerkleTrie

	log.WithFields(logrus.Fields{
		"ChainStartTime": chainStartTime,
	}).Info("Minimum number of validators reached for beacon-chain to start")
	w.chainStartFeed.Send(chainStartTime)
}

// processPastLogs processes all the past logs from the deposit contract and
// updates the deposit trie with the data from each individual log.
func (w *Web3Service) processPastLogs() error {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			w.depositContractAddress,
		},
	}

	logs, err := w.httpLogger.FilterLogs(w.ctx, query)
	if err != nil {
		return err
	}

	for _, log := range logs {
		w.ProcessLog(log)
	}
	w.lastRequestedBlock.Set(w.blockHeight)

	currentState, err := w.beaconDB.HeadState(w.ctx)
	if err != nil {
		return fmt.Errorf("could not get head state: %v", err)
	}
	if currentState != nil && currentState.DepositIndex > 0 {
		w.beaconDB.PrunePendingDeposits(w.ctx, currentState.DepositIndex)
	}

	return nil
}

// requestBatchedLogs requests and processes all the logs from the period
// last polled to now.
func (w *Web3Service) requestBatchedLogs() error {
	// We request for the nth block behind the current head, in order to have
	// stabilized logs when we retrieve it from the 1.0 chain.
	requestedBlock := big.NewInt(0).Sub(w.blockHeight, big.NewInt(params.BeaconConfig().LogBlockDelay))
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			w.depositContractAddress,
		},
		FromBlock: w.lastRequestedBlock.Add(w.lastRequestedBlock, big.NewInt(1)),
		ToBlock:   requestedBlock,
	}
	logs, err := w.httpLogger.FilterLogs(w.ctx, query)
	if err != nil {
		return err
	}

	// Only process log slices which are larger than zero.
	if len(logs) > 0 {
		log.Debug("Processing Batched Logs")
		for _, log := range logs {
			w.ProcessLog(log)
		}
	}

	w.lastRequestedBlock.Set(requestedBlock)
	return nil
}

package powchain

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

var (
	depositEventSignature    = []byte("Deposit(bytes,bytes,bytes,bytes,bytes)")
	chainStartEventSignature = []byte("Eth2Genesis(bytes32,bytes,bytes)")
)

// HasChainStartLogOccurred queries all logs in the deposit contract to verify
// if ChainStart has occurred.
func (w *Web3Service) HasChainStartLogOccurred() (bool, error) {
	return w.depositContractCaller.ChainStarted(&bind.CallOpts{})
}

func (w *Web3Service) ETH2GenesisTime() (uint64, error) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			w.depositContractAddress,
		},
		Topics: [][]common.Hash{{hashutil.Hash(chainStartEventSignature)}},
	}
	logs, err := w.httpLogger.FilterLogs(w.ctx, query)
	if err != nil {
		return 0, err
	}
	if len(logs) == 0 {
		return 0, errors.New("no chainstart logs exist")
	}

	_, _, timestampData, err := contracts.UnpackChainStartLogData(logs[0].Data)
	if err != nil {
		return 0, fmt.Errorf("unable to unpack ChainStart log data %v", err)
	}
	timestamp := binary.LittleEndian.Uint64(timestampData)
	return timestamp, nil
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
	log.WithField("signature", fmt.Sprintf("%#x", depositLog.Topics[0])).Debug("Not a valid event signature")
}

// ProcessDepositLog processes the log which had been received from
// the ETH1.0 chain by trying to ascertain which participant deposited
// in the contract.
func (w *Web3Service) ProcessDepositLog(depositLog gethTypes.Log) {
	pubkey, withdrawalCredentials, amount, signature, merkleTreeIndex, err := contracts.UnpackDepositLogData(depositLog.Data)
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
	depositData := &pb.DepositData{
		Amount:                bytesutil.FromBytes8(amount),
		Pubkey:                pubkey,
		Signature:             signature,
		WithdrawalCredentials: withdrawalCredentials,
	}

	depositHash, err := hashutil.DepositHash(depositData)
	if err != nil {
		log.Errorf("Unable to determine hashed value of deposit %v", err)
		return
	}

	if err := w.depositTrie.InsertIntoTrie(depositHash[:], int(index)); err != nil {
		log.Errorf("Unable to insert deposit into trie %v", err)
		return
	}

	proof, err := w.depositTrie.MerkleProof(int(index))
	if err != nil {
		log.Errorf("Unable to generate merkle proof for deposit %v", err)
		return
	}

	deposit := &pb.Deposit{
		Data:  depositData,
		Index: index,
		Proof: proof,
	}

	// Make sure duplicates are rejected pre-chainstart.
	if !w.chainStarted && validData {
		var pubkey = fmt.Sprintf("#%x", depositData.Pubkey)
		if w.beaconDB.PubkeyInChainstart(w.ctx, pubkey) {
			log.Warnf("Pubkey %#x has already been submitted for chainstart", pubkey)
		} else {
			w.beaconDB.MarkPubkeyForChainstart(w.ctx, pubkey)
		}
	}

	// We always store all historical deposits in the DB.
	w.beaconDB.InsertDeposit(w.ctx, deposit, big.NewInt(int64(depositLog.BlockNumber)))

	if !w.chainStarted {
		w.chainStartDeposits = append(w.chainStartDeposits, deposit)
	} else {
		w.beaconDB.InsertPendingDeposit(w.ctx, deposit, big.NewInt(int64(depositLog.BlockNumber)))
	}
	if validData {
		log.WithFields(logrus.Fields{
			"publicKey":       fmt.Sprintf("%#x", depositData.Pubkey),
			"merkleTreeIndex": index,
		}).Debug("Deposit registered from deposit contract")
		validDepositsCount.Inc()
	} else {
		log.WithFields(logrus.Fields{
			"merkleTreeIndex": index,
		}).Info("Invalid deposit registered in deposit contract")
	}
}

// ProcessChainStartLog processes the log which had been received from
// the ETH1.0 chain by trying to determine when to start the beacon chain.
func (w *Web3Service) ProcessChainStartLog(depositLog gethTypes.Log) {
	chainStartCount.Inc()
	chainStartDepositRoot, _, timestampData, err := contracts.UnpackChainStartLogData(depositLog.Data)
	if err != nil {
		log.Errorf("Unable to unpack ChainStart log data %v", err)
		return
	}

	w.chainStartETH1Data = &pb.Eth1Data{
		BlockRoot:   depositLog.BlockHash[:],
		DepositRoot: chainStartDepositRoot[:],
	}

	timestamp := binary.LittleEndian.Uint64(timestampData)
	w.chainStarted = true
	w.depositRoot = chainStartDepositRoot[:]
	chainStartTime := time.Unix(int64(timestamp), 0)

	depHashes, err := w.ChainStartDepositHashes()
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

// ChainStartDepositHashes returns the hashes of all the chainstart deposits
// stored in memory.
func (w *Web3Service) ChainStartDepositHashes() ([][]byte, error) {
	hashes := make([][]byte, len(w.chainStartDeposits))
	for i, dep := range w.chainStartDeposits {
		hash, err := hashutil.DepositHash(dep.Data)
		if err != nil {
			return nil, err
		}
		hashes[i] = hash[:]
	}
	return hashes, nil
}

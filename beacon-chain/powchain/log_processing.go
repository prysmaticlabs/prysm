package powchain

import (
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

// ETH2GenesisTime retrieves the genesis time and eth1 block number of the beacon chain
// from the deposit contract.
func (w *Web3Service) ETH2GenesisTime() (uint64, *big.Int) {
	return w.eth2GenesisTime, w.chainStartBlockNumber
}

// ProcessLog is the main method which handles the processing of all
// logs from the deposit contract on the ETH1.0 chain.
func (w *Web3Service) ProcessLog(depositLog gethTypes.Log) {
	w.processingLock.Lock()
	defer w.processingLock.Unlock()
	// Process logs according to their event signature.
	if depositLog.Topics[0] == hashutil.HashKeccak256(depositEventSignature) {
		w.ProcessDepositLog(depositLog)
		if !w.chainStarted {
			if depositLog.BlockHash == [32]byte{} {
				log.Error("Got empty blockhash from powchain service")
				return
			}
			blk, err := w.blockFetcher.BlockByHash(w.ctx, depositLog.BlockHash)
			if err != nil {
				log.Errorf("Could not get eth1 block %v", err)
				return
			}
			if blk == nil {
				log.Errorf("Got empty block from powchain service %v", err)
				return
			}
			timeStamp := blk.Time()
			triggered := state.IsValidGenesisState(w.activeValidatorCount, timeStamp)
			if triggered {
				w.setGenesisTime(timeStamp)
				w.ProcessChainStart(uint64(w.eth2GenesisTime), depositLog.BlockHash, blk.Number())
			}
		}
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
		log.Errorf("Could not unpack log: %v", err)
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
	depositData := &ethpb.Deposit_Data{
		Amount:                bytesutil.FromBytes8(amount),
		PublicKey:             pubkey,
		Signature:             signature,
		WithdrawalCredentials: withdrawalCredentials,
	}

	depositHash, err := ssz.HashTreeRoot(depositData)
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

	deposit := &ethpb.Deposit{
		Data:  depositData,
		Proof: proof,
	}

	// Make sure duplicates are rejected pre-chainstart.
	if !w.chainStarted && validData {
		var pubkey = fmt.Sprintf("#%x", depositData.PublicKey)
		if w.beaconDB.PubkeyInChainstart(w.ctx, pubkey) {
			log.Warnf("Pubkey %#x has already been submitted for chainstart", pubkey)
		} else {
			w.beaconDB.MarkPubkeyForChainstart(w.ctx, pubkey)
		}

	}

	// We always store all historical deposits in the DB.
	w.beaconDB.InsertDeposit(w.ctx, deposit, big.NewInt(int64(depositLog.BlockNumber)), int(index), w.depositTrie.Root())

	if !w.chainStarted {
		w.chainStartDeposits = append(w.chainStartDeposits, deposit)
		root := w.depositTrie.Root()
		eth1Data := &ethpb.Eth1Data{
			DepositRoot:  root[:],
			DepositCount: uint64(len(w.chainStartDeposits)),
		}
		if err := w.processDeposit(eth1Data, deposit); err != nil {
			log.Errorf("Invalid deposit processed: %v", err)
			validData = false
		}
	} else {
		w.beaconDB.InsertPendingDeposit(w.ctx, deposit, big.NewInt(int64(depositLog.BlockNumber)), int(index), w.depositTrie.Root())
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
}

// ProcessChainStart processes the log which had been received from
// the ETH1.0 chain by trying to determine when to start the beacon chain.
func (w *Web3Service) ProcessChainStart(genesisTime uint64, eth1BlockHash [32]byte, blockNumber *big.Int) {
	w.chainStarted = true
	w.chainStartBlockNumber = blockNumber

	chainStartTime := time.Unix(int64(genesisTime), 0)
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

	for i := range w.chainStartDeposits {
		proof, err := sparseMerkleTrie.MerkleProof(i)
		if err != nil {
			log.Errorf("Unable to generate deposit proof %v", err)
		}
		w.chainStartDeposits[i].Proof = proof
	}

	w.depositTrie = sparseMerkleTrie
	root := sparseMerkleTrie.Root()
	w.chainStartETH1Data = &ethpb.Eth1Data{
		DepositCount: uint64(len(w.chainStartDeposits)),
		DepositRoot:  root[:],
		BlockHash:    eth1BlockHash[:],
	}

	log.WithFields(logrus.Fields{
		"ChainStartTime": chainStartTime,
	}).Info("Minimum number of validators reached for beacon-chain to start")
	w.chainStartFeed.Send(chainStartTime)
}

func (w *Web3Service) setGenesisTime(timeStamp uint64) {
	if featureconfig.FeatureConfig().NoGenesisDelay {
		w.eth2GenesisTime = uint64(time.Unix(int64(timeStamp), 0).Add(30 * time.Second).Unix())
	} else {
		timeStampRdDown := timeStamp - timeStamp%params.BeaconConfig().SecondsPerDay
		// genesisTime will be set to the first second of the day, two days after it was triggered.
		w.eth2GenesisTime = timeStampRdDown + 2*params.BeaconConfig().SecondsPerDay
	}
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
		return errors.Wrap(err, "could not get head state")
	}

	if currentState != nil && currentState.Eth1DepositIndex > 0 {
		w.beaconDB.PrunePendingDeposits(w.ctx, int(currentState.Eth1DepositIndex))
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
		hash, err := ssz.HashTreeRoot(dep.Data)
		if err != nil {
			return nil, err
		}
		hashes[i] = hash[:]
	}
	return hashes, nil
}

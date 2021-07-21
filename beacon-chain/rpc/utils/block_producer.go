package utils

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"reflect"
	"time"

	fastssz "github.com/ferranbt/fastssz"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/interop"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/proto/interfaces"
	attaggregation "github.com/prysmaticlabs/prysm/shared/aggregation/attestations"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// eth1DataNotification is a latch to stop flooding logs with the same warning.
var eth1DataNotification bool

const eth1dataTimeout = 2 * time.Second

type eth1DataSingleVote struct {
	eth1Data    *ethpb.Eth1Data
	blockHeight *big.Int
}

type eth1DataAggregatedVote struct {
	data  eth1DataSingleVote
	votes int
}

// BlockProducer is responsible for creating beacon blocks.
type BlockProducer interface {
	ProduceBlock(ctx context.Context, slot types.Slot, randaoReveal []byte, graffiti []byte) (*ethpb.BeaconBlock, error)
}

// BlockProvider is a real implementation of BlockProducer
type BlockProvider struct {
	HeadFetcher            blockchain.HeadFetcher
	Eth1InfoFetcher        powchain.ChainInfoFetcher
	Eth1BlockFetcher       powchain.POWBlockFetcher
	DepositFetcher         depositcache.DepositFetcher
	ChainStartFetcher      powchain.ChainStartFetcher
	BlockFetcher           powchain.POWBlockFetcher
	PendingDepositsFetcher depositcache.PendingDepositsFetcher
	AttPool                attestations.Pool
	SlashingsPool          slashings.PoolManager
	ExitPool               voluntaryexits.PoolManager
	StateGen               stategen.StateManager
	MockEth1Votes          bool
}

// ProduceBlock creates an unsigned beacon block from the provided arguments.
func (p *BlockProvider) ProduceBlock(ctx context.Context, slot types.Slot, randaoReveal, graffiti []byte) (*ethpb.BeaconBlock, error) {
	// Retrieve the parent block as the current head of the canonical chain.
	parentRoot, err := p.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Could not retrieve head root")
	}

	head, err := p.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get head state")
	}

	if featureconfig.Get().EnableNextSlotStateCache {
		head, err = state.ProcessSlotsUsingNextSlotCache(ctx, head, parentRoot, slot)
		if err != nil {
			return nil, errors.Wrap(err, "Could not advance slots to calculate proposer index")
		}
	} else {
		head, err = state.ProcessSlots(ctx, head, slot)
		if err != nil {
			return nil, errors.Wrap(err, "Could not advance slot to calculate proposer index")
		}
	}

	eth1Data, err := p.eth1DataMajorityVote(ctx, head)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get ETH1 data")
	}

	// Pack ETH1 deposits which have not been included in the beacon chain.
	deposits, err := p.deposits(ctx, head, eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get ETH1 deposits")
	}

	// Pack aggregated attestations which have not been included in the beacon chain.
	atts, err := p.packAttestations(ctx, head)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get attestations to pack into block")
	}

	// Use zero hash as stub for state root to compute later.
	stateRoot := params.BeaconConfig().ZeroHash[:]

	graf := bytesutil.ToBytes32(graffiti)

	// Calculate new proposer index.
	idx, err := helpers.BeaconProposerIndex(head)
	if err != nil {
		return nil, errors.Wrap(err, "Could not calculate proposer index")
	}

	blk := &ethpb.BeaconBlock{
		Slot:          slot,
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		ProposerIndex: idx,
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:          eth1Data,
			Deposits:          deposits,
			Attestations:      atts,
			RandaoReveal:      randaoReveal,
			ProposerSlashings: p.SlashingsPool.PendingProposerSlashings(ctx, head, false /*noLimit*/),
			AttesterSlashings: p.SlashingsPool.PendingAttesterSlashings(ctx, head, false /*noLimit*/),
			VoluntaryExits:    p.ExitPool.PendingExits(head, slot, false /*noLimit*/),
			Graffiti:          graf[:],
		},
	}

	// Compute state root with the newly constructed block.
	stateRoot, err = p.computeStateRoot(ctx, wrapper.WrappedPhase0SignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: blk, Signature: make([]byte, 96)}))
	if err != nil {
		interop.WriteBlockToDisk(wrapper.WrappedPhase0SignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: blk}), true /*failed*/)
		return nil, errors.Wrap(err, "Could not compute state root")
	}
	blk.StateRoot = stateRoot

	return blk, nil
}

// eth1DataMajorityVote determines the appropriate eth1data for a block proposal using
// an algorithm called Voting with the Majority. The algorithm works as follows:
//  - Determine the timestamp for the start slot for the eth1 voting period.
//  - Determine the earliest and latest timestamps that a valid block can have.
//  - Determine the first block not before the earliest timestamp. This block is the lower bound.
//  - Determine the last block not after the latest timestamp. This block is the upper bound.
//  - If the last block is too early, use current eth1data from the beacon state.
//  - Filter out votes on unknown blocks and blocks which are outside of the range determined by the lower and upper bounds.
//  - If no blocks are left after filtering votes, use eth1data from the latest valid block.
//  - Otherwise:
//    - Determine the vote with the highest count. Prefer the vote with the highest eth1 block height in the event of a tie.
//    - This vote's block is the eth1 block to use for the block proposal.
func (p *BlockProvider) eth1DataMajorityVote(ctx context.Context, beaconState iface.BeaconState) (*ethpb.Eth1Data, error) {
	ctx, cancel := context.WithTimeout(ctx, eth1dataTimeout)
	defer cancel()

	slot := beaconState.Slot()
	votingPeriodStartTime := p.slotStartTime(slot)

	if p.MockEth1Votes {
		return p.mockETH1DataVote(ctx, slot)
	}
	if !p.Eth1InfoFetcher.IsConnectedToETH1() {
		return p.randomETH1DataVote(ctx)
	}
	eth1DataNotification = false

	eth1FollowDistance := params.BeaconConfig().Eth1FollowDistance
	earliestValidTime := votingPeriodStartTime - 2*params.BeaconConfig().SecondsPerETH1Block*eth1FollowDistance
	latestValidTime := votingPeriodStartTime - params.BeaconConfig().SecondsPerETH1Block*eth1FollowDistance

	lastBlockByEarliestValidTime, err := p.Eth1BlockFetcher.BlockByTimestamp(ctx, earliestValidTime)
	if err != nil {
		log.WithError(err).Error("Could not get last block by earliest valid time")
		return p.randomETH1DataVote(ctx)
	}
	// Increment the earliest block if the original block's time is before valid time.
	// This is very likely to happen because BlockTimeByHeight returns the last block AT OR BEFORE the specified time.
	if lastBlockByEarliestValidTime.Time < earliestValidTime {
		lastBlockByEarliestValidTime.Number = big.NewInt(0).Add(lastBlockByEarliestValidTime.Number, big.NewInt(1))
	}

	lastBlockByLatestValidTime, err := p.Eth1BlockFetcher.BlockByTimestamp(ctx, latestValidTime)
	if err != nil {
		log.WithError(err).Error("Could not get last block by latest valid time")
		return p.randomETH1DataVote(ctx)
	}
	if lastBlockByLatestValidTime.Time < earliestValidTime {
		return p.HeadFetcher.HeadETH1Data(), nil
	}

	lastBlockDepositCount, lastBlockDepositRoot := p.DepositFetcher.DepositsNumberAndRootAtHeight(ctx, lastBlockByLatestValidTime.Number)
	if lastBlockDepositCount == 0 {
		return p.ChainStartFetcher.ChainStartEth1Data(), nil
	}

	if lastBlockDepositCount >= p.HeadFetcher.HeadETH1Data().DepositCount {
		hash, err := p.Eth1BlockFetcher.BlockHashByHeight(ctx, lastBlockByLatestValidTime.Number)
		if err != nil {
			log.WithError(err).Error("Could not get hash of last block by latest valid time")
			return p.randomETH1DataVote(ctx)
		}
		return &ethpb.Eth1Data{
			BlockHash:    hash.Bytes(),
			DepositCount: lastBlockDepositCount,
			DepositRoot:  lastBlockDepositRoot[:],
		}, nil
	}
	return p.HeadFetcher.HeadETH1Data(), nil
}

func (p *BlockProvider) slotStartTime(slot types.Slot) uint64 {
	startTime, _ := p.Eth1InfoFetcher.Eth2GenesisPowchainInfo()
	return helpers.VotingPeriodStartTime(startTime, slot)
}

func (p *BlockProvider) inRangeVotes(ctx context.Context,
	beaconState iface.ReadOnlyBeaconState,
	firstValidBlockNumber, lastValidBlockNumber *big.Int) ([]eth1DataSingleVote, error) {

	currentETH1Data := p.HeadFetcher.HeadETH1Data()

	var inRangeVotes []eth1DataSingleVote
	for _, eth1Data := range beaconState.Eth1DataVotes() {
		exists, height, err := p.BlockFetcher.BlockExistsWithCache(ctx, bytesutil.ToBytes32(eth1Data.BlockHash))
		if err != nil {
			log.Warningf("Could not fetch eth1data height for received eth1data vote: %v", err)
		}
		// Make sure we don't "undo deposit progress". See https://github.com/ethereum/eth2.0-specs/pull/1836
		if eth1Data.DepositCount < currentETH1Data.DepositCount {
			continue
		}
		// firstValidBlockNumber.Cmp(height) < 1 filters out all blocks before firstValidBlockNumber
		// lastValidBlockNumber.Cmp(height) > -1 filters out all blocks after lastValidBlockNumber
		// These filters result in the range [firstValidBlockNumber, lastValidBlockNumber]
		if exists && firstValidBlockNumber.Cmp(height) < 1 && lastValidBlockNumber.Cmp(height) > -1 {
			inRangeVotes = append(inRangeVotes, eth1DataSingleVote{eth1Data: eth1Data, blockHeight: height})
		}
	}

	return inRangeVotes, nil
}

func chosenEth1DataMajorityVote(votes []eth1DataSingleVote) eth1DataAggregatedVote {
	var voteCount []eth1DataAggregatedVote
	for _, singleVote := range votes {
		newVote := true
		for i, aggregatedVote := range voteCount {
			aggregatedData := aggregatedVote.data
			if reflect.DeepEqual(singleVote.eth1Data, aggregatedData.eth1Data) {
				voteCount[i].votes++
				newVote = false
				break
			}
		}

		if newVote {
			voteCount = append(voteCount, eth1DataAggregatedVote{data: singleVote, votes: 1})
		}
	}
	if len(voteCount) == 0 {
		return eth1DataAggregatedVote{}
	}
	currentVote := voteCount[0]
	for _, aggregatedVote := range voteCount[1:] {
		// Choose new eth1data if it has more votes or the same number of votes with a bigger block height.
		if aggregatedVote.votes > currentVote.votes ||
			(aggregatedVote.votes == currentVote.votes &&
				aggregatedVote.data.blockHeight.Cmp(currentVote.data.blockHeight) == 1) {
			currentVote = aggregatedVote
		}
	}

	return currentVote
}

func (p *BlockProvider) mockETH1DataVote(ctx context.Context, slot types.Slot) (*ethpb.Eth1Data, error) {
	if !eth1DataNotification {
		log.Warn("Beacon Node is no longer connected to an ETH1 chain, so ETH1 data votes are now mocked.")
		eth1DataNotification = true
	}
	// If a mock eth1 data votes is specified, we use the following for the
	// eth1data we provide to every proposer based on https://github.com/ethereum/eth2.0-pm/issues/62:
	//
	// slot_in_voting_period = current_slot % SLOTS_PER_ETH1_VOTING_PERIOD
	// Eth1Data(
	//   DepositRoot = hash(current_epoch + slot_in_voting_period),
	//   DepositCount = state.eth1_deposit_index,
	//   BlockHash = hash(hash(current_epoch + slot_in_voting_period)),
	// )
	slotInVotingPeriod := slot.ModSlot(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod)))
	headState, err := p.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, err
	}
	var enc []byte
	enc = fastssz.MarshalUint64(enc, uint64(helpers.SlotToEpoch(slot))+uint64(slotInVotingPeriod))
	depRoot := hashutil.Hash(enc)
	blockHash := hashutil.Hash(depRoot[:])
	return &ethpb.Eth1Data{
		DepositRoot:  depRoot[:],
		DepositCount: headState.Eth1DepositIndex(),
		BlockHash:    blockHash[:],
	}, nil
}

func (p *BlockProvider) randomETH1DataVote(ctx context.Context) (*ethpb.Eth1Data, error) {
	if !eth1DataNotification {
		log.Warn("Beacon Node is no longer connected to an ETH1 chain, so ETH1 data votes are now random.")
		eth1DataNotification = true
	}
	headState, err := p.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, err
	}

	// set random roots and block hashes to prevent a majority from being
	// built if the eth1 node is offline
	randGen := rand.NewGenerator()
	depRoot := hashutil.Hash(bytesutil.Bytes32(randGen.Uint64()))
	blockHash := hashutil.Hash(bytesutil.Bytes32(randGen.Uint64()))
	return &ethpb.Eth1Data{
		DepositRoot:  depRoot[:],
		DepositCount: headState.Eth1DepositIndex(),
		BlockHash:    blockHash[:],
	}, nil
}

// computeStateRoot computes the state root after a block has been processed through a state transition and
// returns it to the validator client.
func (p *BlockProvider) computeStateRoot(ctx context.Context, block interfaces.SignedBeaconBlock) ([]byte, error) {
	beaconState, err := p.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(block.Block().ParentRoot()))
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve beacon state")
	}
	root, err := state.CalculateStateRoot(
		ctx,
		beaconState,
		block,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not calculate state root at slot %d", beaconState.Slot())
	}

	log.WithField("beaconStateRoot", fmt.Sprintf("%#x", root)).Debugf("Computed state root")
	return root[:], nil
}

// deposits returns a list of pending deposits that are ready for inclusion in the next beacon
// block. Determining deposits depends on the current eth1data vote for the block and whether or not
// this eth1data has enough support to be considered for deposits inclusion. If current vote has
// enough support, then use that vote for basis of determining deposits, otherwise use current state
// eth1data.
func (p *BlockProvider) deposits(
	ctx context.Context,
	beaconState iface.BeaconState,
	currentVote *ethpb.Eth1Data,
) ([]*ethpb.Deposit, error) {
	ctx, span := trace.StartSpan(ctx, "BlockProvider.deposits")
	defer span.End()

	if p.MockEth1Votes || !p.Eth1InfoFetcher.IsConnectedToETH1() {
		return []*ethpb.Deposit{}, nil
	}
	// Need to fetch if the deposits up to the state's latest eth 1 data matches
	// the number of all deposits in this RPC call. If not, then we return nil.
	canonicalEth1Data, canonicalEth1DataHeight, err := p.canonicalEth1Data(ctx, beaconState, currentVote)
	if err != nil {
		return nil, err
	}

	_, genesisEth1Block := p.Eth1InfoFetcher.Eth2GenesisPowchainInfo()
	if genesisEth1Block.Cmp(canonicalEth1DataHeight) == 0 {
		return []*ethpb.Deposit{}, nil
	}

	// If there are no pending deposits, exit early.
	allPendingContainers := p.PendingDepositsFetcher.PendingContainers(ctx, canonicalEth1DataHeight)
	if len(allPendingContainers) == 0 {
		return []*ethpb.Deposit{}, nil
	}

	depositTrie, err := p.depositTrie(ctx, canonicalEth1Data, canonicalEth1DataHeight)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve deposit trie")
	}

	// Deposits need to be received in order of merkle index root, so this has to make sure
	// deposits are sorted from lowest to highest.
	var pendingDeps []*dbpb.DepositContainer
	for _, dep := range allPendingContainers {
		if uint64(dep.Index) >= beaconState.Eth1DepositIndex() && uint64(dep.Index) < canonicalEth1Data.DepositCount {
			pendingDeps = append(pendingDeps, dep)
		}
	}

	for i := range pendingDeps {
		// Don't construct merkle proof if the number of deposits is more than max allowed in block.
		if uint64(i) == params.BeaconConfig().MaxDeposits {
			break
		}
		pendingDeps[i].Deposit, err = constructMerkleProof(depositTrie, int(pendingDeps[i].Index), pendingDeps[i].Deposit)
		if err != nil {
			return nil, err
		}
	}
	// Limit the return of pending deposits to not be more than max deposits allowed in block.
	var pendingDeposits []*ethpb.Deposit
	for i := uint64(0); i < uint64(len(pendingDeps)) && i < params.BeaconConfig().MaxDeposits; i++ {
		pendingDeposits = append(pendingDeposits, pendingDeps[i].Deposit)
	}
	return pendingDeposits, nil
}

// canonicalEth1Data determines the canonical eth1data and eth1 block height to use for determining deposits.
func (p *BlockProvider) canonicalEth1Data(
	ctx context.Context,
	beaconState iface.BeaconState,
	currentVote *ethpb.Eth1Data) (*ethpb.Eth1Data, *big.Int, error) {

	var eth1BlockHash [32]byte

	// Add in current vote, to get accurate vote tally
	if err := beaconState.AppendEth1DataVotes(currentVote); err != nil {
		return nil, nil, errors.Wrap(err, "could not append eth1 data votes to state")
	}
	hasSupport, err := blocks.Eth1DataHasEnoughSupport(beaconState, currentVote)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not determine if current eth1data vote has enough support")
	}
	var canonicalEth1Data *ethpb.Eth1Data
	if hasSupport {
		canonicalEth1Data = currentVote
		eth1BlockHash = bytesutil.ToBytes32(currentVote.BlockHash)
	} else {
		canonicalEth1Data = beaconState.Eth1Data()
		eth1BlockHash = bytesutil.ToBytes32(beaconState.Eth1Data().BlockHash)
	}
	_, canonicalEth1DataHeight, err := p.Eth1BlockFetcher.BlockExists(ctx, eth1BlockHash)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not fetch eth1data height")
	}
	return canonicalEth1Data, canonicalEth1DataHeight, nil
}

func (p *BlockProvider) depositTrie(ctx context.Context, canonicalEth1Data *ethpb.Eth1Data, canonicalEth1DataHeight *big.Int) (*trieutil.SparseMerkleTrie, error) {
	ctx, span := trace.StartSpan(ctx, "BlockProvider.depositTrie")
	defer span.End()

	var depositTrie *trieutil.SparseMerkleTrie

	finalizedDeposits := p.DepositFetcher.FinalizedDeposits(ctx)
	depositTrie = finalizedDeposits.Deposits
	upToEth1DataDeposits := p.DepositFetcher.NonFinalizedDeposits(ctx, canonicalEth1DataHeight)
	insertIndex := finalizedDeposits.MerkleTrieIndex + 1

	for _, dep := range upToEth1DataDeposits {
		depHash, err := dep.Data.HashTreeRoot()
		if err != nil {
			return nil, errors.Wrap(err, "could not hash deposit data")
		}
		depositTrie.Insert(depHash[:], int(insertIndex))
		insertIndex++
	}
	valid, err := p.validateDepositTrie(depositTrie, canonicalEth1Data)
	// Log a warning here, as the cached trie is invalid.
	if !valid {
		log.Warnf("Cached deposit trie is invalid, rebuilding it now: %v", err)
		return p.rebuildDepositTrie(ctx, canonicalEth1Data, canonicalEth1DataHeight)
	}

	return depositTrie, nil
}

// rebuilds our deposit trie by recreating it from all processed deposits till
// specified eth1 block height.
func (p *BlockProvider) rebuildDepositTrie(ctx context.Context, canonicalEth1Data *ethpb.Eth1Data, canonicalEth1DataHeight *big.Int) (*trieutil.SparseMerkleTrie, error) {
	ctx, span := trace.StartSpan(ctx, "BlockProvider.rebuildDepositTrie")
	defer span.End()

	deposits := p.DepositFetcher.AllDeposits(ctx, canonicalEth1DataHeight)
	trieItems := make([][]byte, 0, len(deposits))
	for _, dep := range deposits {
		depHash, err := dep.Data.HashTreeRoot()
		if err != nil {
			return nil, errors.Wrap(err, "could not hash deposit data")
		}
		trieItems = append(trieItems, depHash[:])
	}
	depositTrie, err := trieutil.GenerateTrieFromItems(trieItems, params.BeaconConfig().DepositContractTreeDepth)
	if err != nil {
		return nil, err
	}

	valid, err := p.validateDepositTrie(depositTrie, canonicalEth1Data)
	// Log an error here, as even with rebuilding the trie, it is still invalid.
	if !valid {
		log.Errorf("Rebuilt deposit trie is invalid: %v", err)
	}
	return depositTrie, nil
}

// validate that the provided deposit trie matches up with the canonical eth1 data provided.
func (p *BlockProvider) validateDepositTrie(trie *trieutil.SparseMerkleTrie, canonicalEth1Data *ethpb.Eth1Data) (bool, error) {
	if trie.NumOfItems() != int(canonicalEth1Data.DepositCount) {
		return false, errors.Errorf("wanted the canonical count of %d but received %d", canonicalEth1Data.DepositCount, trie.NumOfItems())
	}
	rt := trie.HashTreeRoot()
	if !bytes.Equal(rt[:], canonicalEth1Data.DepositRoot) {
		return false, errors.Errorf("wanted the canonical deposit root of %#x but received %#x", canonicalEth1Data.DepositRoot, rt)
	}
	return true, nil
}

func constructMerkleProof(trie *trieutil.SparseMerkleTrie, index int, deposit *ethpb.Deposit) (*ethpb.Deposit, error) {
	proof, err := trie.MerkleProof(index)
	if err != nil {
		return nil, errors.Wrapf(err, "could not generate merkle proof for deposit at index %d", index)
	}
	// For every deposit, we construct a Merkle proof using the powchain service's
	// in-memory deposits trie, which is updated only once the state's LatestETH1Data
	// property changes during a state transition after a voting period.
	deposit.Proof = proof
	return deposit, nil
}

func (p *BlockProvider) packAttestations(ctx context.Context, latestState iface.BeaconState) ([]*ethpb.Attestation, error) {
	ctx, span := trace.StartSpan(ctx, "BlockProvider.packAttestations")
	defer span.End()

	atts := p.AttPool.AggregatedAttestations()
	atts, err := p.filterAttestationsForBlockInclusion(ctx, latestState, atts)
	if err != nil {
		return nil, errors.Wrap(err, "could not filter attestations")
	}

	// If there is any room left in the block, consider unaggregated attestations as well.
	numAtts := uint64(len(atts))
	if numAtts < params.BeaconConfig().MaxAttestations {
		uAtts, err := p.AttPool.UnaggregatedAttestations()
		if err != nil {
			return nil, errors.Wrap(err, "could not get unaggregated attestations")
		}
		uAtts, err = p.filterAttestationsForBlockInclusion(ctx, latestState, uAtts)
		if err != nil {
			return nil, errors.Wrap(err, "could not filter attestations")
		}
		atts = append(atts, uAtts...)

		attsByDataRoot := make(map[[32]byte][]*ethpb.Attestation, len(atts))
		for _, att := range atts {
			attDataRoot, err := att.Data.HashTreeRoot()
			if err != nil {
				return nil, err
			}
			attsByDataRoot[attDataRoot] = append(attsByDataRoot[attDataRoot], att)
		}

		attsForInclusion := producerAtts(make([]*ethpb.Attestation, 0))
		for _, as := range attsByDataRoot {
			as, err := attaggregation.Aggregate(as)
			if err != nil {
				return nil, err
			}
			attsForInclusion = append(attsForInclusion, as...)
		}
		deduped, err := attsForInclusion.dedup()
		if err != nil {
			return nil, err
		}
		sorted, err := deduped.sortByProfitability()
		if err != nil {
			return nil, err
		}
		atts = sorted.limitToMaxAttestations()
	}
	return atts, nil
}

// This filters the input attestations to return a list of valid attestations to be packaged inside a beacon block.
func (p *BlockProvider) filterAttestationsForBlockInclusion(ctx context.Context, st iface.BeaconState, atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	ctx, span := trace.StartSpan(ctx, "BlockProvider.filterAttestationsForBlockInclusion")
	defer span.End()

	validAtts, invalidAtts := producerAtts(atts).filter(ctx, st)
	if err := p.deleteAttsInPool(ctx, invalidAtts); err != nil {
		return nil, err
	}
	deduped, err := validAtts.dedup()
	if err != nil {
		return nil, err
	}
	sorted, err := deduped.sortByProfitability()
	if err != nil {
		return nil, err
	}
	return sorted.limitToMaxAttestations(), nil
}

// The input attestations are processed and seen by the node, this deletes them from pool
// so proposers don't include them in a block for the future.
func (p *BlockProvider) deleteAttsInPool(ctx context.Context, atts []*ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "BlockProvider.deleteAttsInPool")
	defer span.End()

	for _, att := range atts {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if helpers.IsAggregated(att) {
			if err := p.AttPool.DeleteAggregatedAttestation(att); err != nil {
				return err
			}
		} else {
			if err := p.AttPool.DeleteUnaggregatedAttestation(att); err != nil {
				return err
			}
		}
	}
	return nil
}

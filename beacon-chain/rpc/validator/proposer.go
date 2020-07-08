package validator

import (
	"context"
	"fmt"
	"math/big"
	"time"

	fastssz "github.com/ferranbt/fastssz"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/interop"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// eth1DataNotification is a latch to stop flooding logs with the same warning.
var eth1DataNotification bool

const eth1dataTimeout = 2 * time.Second

// GetBlock is called by a proposer during its assigned slot to request a block to sign
// by passing in the slot and the signed randao reveal of the slot.
func (vs *Server) GetBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.GetBlock")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(req.Slot)))

	if vs.SyncChecker.Syncing() {
		return nil, status.Errorf(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	// Retrieve the parent block as the current head of the canonical chain.
	parentRoot, err := vs.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve head root: %v", err)
	}
	eth1Data, err := vs.eth1Data(ctx, req.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get ETH1 data: %v", err)
	}

	// Pack ETH1 deposits which have not been included in the beacon chain.
	deposits, err := vs.deposits(ctx, eth1Data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get ETH1 deposits: %v", err)
	}
	head, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state %v", err)
	}
	head, err = state.ProcessSlots(ctx, head, req.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not advance slot to calculate proposer index: %v", err)
	}

	// Pack aggregated attestations which have not been included in the beacon chain.
	atts, err := vs.packAttestations(ctx, head)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get attestations to pack into block: %v", err)
	}

	// Use zero hash as stub for state root to compute later.
	stateRoot := params.BeaconConfig().ZeroHash[:]

	graffiti := bytesutil.ToBytes32(req.Graffiti)

	// Calculate new proposer index.
	idx, err := helpers.BeaconProposerIndex(head)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not calculate proposer index %v", err)
	}

	blk := &ethpb.BeaconBlock{
		Slot:          req.Slot,
		ParentRoot:    parentRoot[:],
		StateRoot:     stateRoot,
		ProposerIndex: idx,
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:          eth1Data,
			Deposits:          deposits,
			Attestations:      atts,
			RandaoReveal:      req.RandaoReveal,
			ProposerSlashings: vs.SlashingsPool.PendingProposerSlashings(ctx, head),
			AttesterSlashings: vs.SlashingsPool.PendingAttesterSlashings(ctx, head),
			VoluntaryExits:    vs.ExitPool.PendingExits(head, req.Slot),
			Graffiti:          graffiti[:],
		},
	}

	// Compute state root with the newly constructed block.
	stateRoot, err = vs.computeStateRoot(ctx, &ethpb.SignedBeaconBlock{Block: blk, Signature: make([]byte, 96)})
	if err != nil {
		interop.WriteBlockToDisk(&ethpb.SignedBeaconBlock{Block: blk}, true /*failed*/)
		return nil, status.Errorf(codes.Internal, "Could not compute state root: %v", err)
	}
	blk.StateRoot = stateRoot

	return blk, nil
}

// ProposeBlock is called by a proposer during its assigned slot to create a block in an attempt
// to get it processed by the beacon node as the canonical head.
func (vs *Server) ProposeBlock(ctx context.Context, blk *ethpb.SignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	root, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not tree hash block: %v", err)
	}

	// Do not block proposal critical path with debug logging or block feed updates.
	defer func() {
		log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(root[:]))).Debugf(
			"Block proposal received via RPC")
		vs.BlockNotifier.BlockFeed().Send(&feed.Event{
			Type: blockfeed.ReceivedBlock,
			Data: &blockfeed.ReceivedBlockData{SignedBlock: blk},
		})
	}()

	if err := vs.BlockReceiver.ReceiveBlock(ctx, blk, root); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not process beacon block: %v", err)
	}

	if err := vs.deleteAttsInPool(ctx, blk.Block.Body.Attestations); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not delete attestations in pool: %v", err)
	}

	return &ethpb.ProposeResponse{
		BlockRoot: root[:],
	}, nil
}

// eth1Data determines the appropriate eth1data for a block proposal. The algorithm for this method
// is as follows:
//  - Determine the timestamp for the start slot for the eth1 voting period.
//  - Determine the most recent eth1 block before that timestamp.
//  - Subtract that eth1block.number by ETH1_FOLLOW_DISTANCE.
//  - This is the eth1block to use for the block proposal.
func (vs *Server) eth1Data(ctx context.Context, slot uint64) (*ethpb.Eth1Data, error) {
	ctx, cancel := context.WithTimeout(ctx, eth1dataTimeout)
	defer cancel()

	if vs.MockEth1Votes {
		return vs.mockETH1DataVote(ctx, slot)
	}

	if !vs.Eth1InfoFetcher.IsConnectedToETH1() {
		return vs.randomETH1DataVote(ctx)
	}
	eth1DataNotification = false

	eth1VotingPeriodStartTime, _ := vs.Eth1InfoFetcher.Eth2GenesisPowchainInfo()
	eth1VotingPeriodStartTime += (slot - (slot % (params.BeaconConfig().EpochsPerEth1VotingPeriod * params.BeaconConfig().SlotsPerEpoch))) * params.BeaconConfig().SecondsPerSlot

	// Look up most recent block up to timestamp
	blockNumber, err := vs.Eth1BlockFetcher.BlockNumberByTimestamp(ctx, eth1VotingPeriodStartTime)
	if err != nil {
		log.WithError(err).Error("Failed to get block number from timestamp")
		return vs.randomETH1DataVote(ctx)
	}
	eth1Data, err := vs.defaultEth1DataResponse(ctx, blockNumber)
	if err != nil {
		log.WithError(err).Error("Failed to get eth1 data from block number")
		return vs.randomETH1DataVote(ctx)
	}

	return eth1Data, nil
}

func (vs *Server) mockETH1DataVote(ctx context.Context, slot uint64) (*ethpb.Eth1Data, error) {
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
	slotInVotingPeriod := slot % (params.BeaconConfig().EpochsPerEth1VotingPeriod * params.BeaconConfig().SlotsPerEpoch)
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, err
	}
	var enc []byte
	enc = fastssz.MarshalUint64(enc, helpers.SlotToEpoch(slot)+slotInVotingPeriod)
	depRoot := hashutil.Hash(enc)
	blockHash := hashutil.Hash(depRoot[:])
	return &ethpb.Eth1Data{
		DepositRoot:  depRoot[:],
		DepositCount: headState.Eth1DepositIndex(),
		BlockHash:    blockHash[:],
	}, nil
}

func (vs *Server) randomETH1DataVote(ctx context.Context) (*ethpb.Eth1Data, error) {
	if !eth1DataNotification {
		log.Warn("Beacon Node is no longer connected to an ETH1 chain, so ETH1 data votes are now random.")
		eth1DataNotification = true
	}
	headState, err := vs.HeadFetcher.HeadState(ctx)
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
func (vs *Server) computeStateRoot(ctx context.Context, block *ethpb.SignedBeaconBlock) ([]byte, error) {
	beaconState, err := vs.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(block.Block.ParentRoot))
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
func (vs *Server) deposits(ctx context.Context, currentVote *ethpb.Eth1Data) ([]*ethpb.Deposit, error) {
	if vs.MockEth1Votes || !vs.Eth1InfoFetcher.IsConnectedToETH1() {
		return []*ethpb.Deposit{}, nil
	}
	// Need to fetch if the deposits up to the state's latest eth 1 data matches
	// the number of all deposits in this RPC call. If not, then we return nil.
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	canonicalEth1Data, canonicalEth1DataHeight, err := vs.canonicalEth1Data(ctx, headState, currentVote)
	if err != nil {
		return nil, err
	}

	_, genesisEth1Block := vs.Eth1InfoFetcher.Eth2GenesisPowchainInfo()
	if genesisEth1Block.Cmp(canonicalEth1DataHeight) == 0 {
		return []*ethpb.Deposit{}, nil
	}

	// If there are no pending deposits, exit early.
	allPendingContainers := vs.PendingDepositsFetcher.PendingContainers(ctx, canonicalEth1DataHeight)
	if len(allPendingContainers) == 0 {
		return []*ethpb.Deposit{}, nil
	}

	depositTrie, err := vs.depositTrie(ctx, canonicalEth1DataHeight)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve deposit trie")
	}

	// Deposits need to be received in order of merkle index root, so this has to make sure
	// deposits are sorted from lowest to highest.
	var pendingDeps []*dbpb.DepositContainer
	for _, dep := range allPendingContainers {
		if uint64(dep.Index) >= headState.Eth1DepositIndex() && uint64(dep.Index) < canonicalEth1Data.DepositCount {
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
func (vs *Server) canonicalEth1Data(ctx context.Context, beaconState *stateTrie.BeaconState, currentVote *ethpb.Eth1Data) (*ethpb.Eth1Data, *big.Int, error) {
	var eth1BlockHash [32]byte

	// Add in current vote, to get accurate vote tally
	if err := beaconState.AppendEth1DataVotes(currentVote); err != nil {
		return nil, nil, errors.Wrap(err, "failed to append eth1 data votes to state")
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
	_, canonicalEth1DataHeight, err := vs.Eth1BlockFetcher.BlockExists(ctx, eth1BlockHash)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not fetch eth1data height")
	}
	return canonicalEth1Data, canonicalEth1DataHeight, nil
}

func (vs *Server) depositTrie(ctx context.Context, canonicalEth1DataHeight *big.Int) (*trieutil.SparseMerkleTrie, error) {
	var depositTrie *trieutil.SparseMerkleTrie

	var finalizedDeposits *depositcache.FinalizedDeposits
	if featureconfig.Get().EnableFinalizedDepositsCache {
		finalizedDeposits = vs.DepositFetcher.FinalizedDeposits(ctx)
	}

	if finalizedDeposits != nil {
		depositTrie = finalizedDeposits.Deposits

		upToEth1DataDeposits := vs.DepositFetcher.NonFinalizedDeposits(ctx, canonicalEth1DataHeight)
		insertIndex := finalizedDeposits.MerkleTrieIndex + 1
		for _, dep := range upToEth1DataDeposits {
			depHash, err := ssz.HashTreeRoot(dep.Data)
			if err != nil {
				return nil, errors.Wrap(err, "could not hash deposit data")
			}
			depositTrie.Insert(depHash[:], int(insertIndex))
			insertIndex++
		}
	} else {
		upToEth1DataDeposits := vs.DepositFetcher.AllDeposits(ctx, canonicalEth1DataHeight)
		depositData := [][]byte{}
		for _, dep := range upToEth1DataDeposits {
			depHash, err := ssz.HashTreeRoot(dep.Data)
			if err != nil {
				return nil, errors.Wrap(err, "could not hash deposit data")
			}
			depositData = append(depositData, depHash[:])
		}

		var err error
		depositTrie, err = trieutil.GenerateTrieFromItems(depositData, int(params.BeaconConfig().DepositContractTreeDepth))
		if err != nil {
			return nil, errors.Wrap(err, "could not generate historical deposit trie from deposits")
		}
	}

	return depositTrie, nil
}

// in case no vote for new eth1data vote considered best vote we
// default into returning the latest deposit root and the block
// hash of eth1 block hash that is FOLLOW_DISTANCE back from its
// latest block.
func (vs *Server) defaultEth1DataResponse(ctx context.Context, currentHeight *big.Int) (*ethpb.Eth1Data, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	eth1FollowDistance := int64(params.BeaconConfig().Eth1FollowDistance)
	ancestorHeight := big.NewInt(0).Sub(currentHeight, big.NewInt(eth1FollowDistance))
	blockHash, err := vs.Eth1BlockFetcher.BlockHashByHeight(ctx, ancestorHeight)
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch ETH1_FOLLOW_DISTANCE ancestor")
	}
	// Fetch all historical deposits up to an ancestor height.
	depositsTillHeight, depositRoot := vs.DepositFetcher.DepositsNumberAndRootAtHeight(ctx, ancestorHeight)
	if depositsTillHeight == 0 {
		return vs.ChainStartFetcher.ChainStartEth1Data(), nil
	}
	// Check for the validity of deposit count.
	currentETH1Data := vs.HeadFetcher.HeadETH1Data()
	if depositsTillHeight < currentETH1Data.DepositCount {
		return currentETH1Data, nil
	}
	return &ethpb.Eth1Data{
		DepositRoot:  depositRoot[:],
		BlockHash:    blockHash[:],
		DepositCount: depositsTillHeight,
	}, nil
}

// This filters the input attestations to return a list of valid attestations to be packaged inside a beacon block.
func (vs *Server) filterAttestationsForBlockInclusion(ctx context.Context, state *stateTrie.BeaconState, atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.filterAttestationsForBlockInclusion")
	defer span.End()

	validAtts := make([]*ethpb.Attestation, 0, len(atts))
	inValidAtts := make([]*ethpb.Attestation, 0, len(atts))

	for i, att := range atts {
		if uint64(i) == params.BeaconConfig().MaxAttestations {
			break
		}

		if _, err := blocks.ProcessAttestation(ctx, state, att); err != nil {
			inValidAtts = append(inValidAtts, att)
			continue

		}
		validAtts = append(validAtts, att)
	}

	if err := vs.deleteAttsInPool(ctx, inValidAtts); err != nil {
		return nil, err
	}

	return validAtts, nil
}

// The input attestations are processed and seen by the node, this deletes them from pool
// so proposers don't include them in a block for the future.
func (vs *Server) deleteAttsInPool(ctx context.Context, atts []*ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.deleteAttsInPool")
	defer span.End()

	for _, att := range atts {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if helpers.IsAggregated(att) {
			if err := vs.AttPool.DeleteAggregatedAttestation(att); err != nil {
				return err
			}
		} else {
			if err := vs.AttPool.DeleteUnaggregatedAttestation(att); err != nil {
				return err
			}
		}
	}
	return nil
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

func (vs *Server) packAttestations(ctx context.Context, latestState *stateTrie.BeaconState) ([]*ethpb.Attestation, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.packAttestations")
	defer span.End()

	atts := vs.AttPool.AggregatedAttestations()
	atts, err := vs.filterAttestationsForBlockInclusion(ctx, latestState, atts)
	if err != nil {
		return nil, errors.Wrap(err, "could not filter attestations")
	}

	// If there is any room left in the block, consider unaggregated attestations as well.
	numAtts := uint64(len(atts))
	if numAtts < params.BeaconConfig().MaxAttestations {
		uAtts := vs.AttPool.UnaggregatedAttestations()
		uAtts, err = vs.filterAttestationsForBlockInclusion(ctx, latestState, uAtts)
		numUAtts := uint64(len(uAtts))
		if numUAtts+numAtts > params.BeaconConfig().MaxAttestations {
			uAtts = uAtts[:params.BeaconConfig().MaxAttestations-numAtts]
		}
		atts = append(atts, uAtts...)
	}
	return atts, nil
}

package rpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/blockutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

// BeaconServer defines a server implementation of the gRPC Beacon service,
// providing RPC endpoints for obtaining the canonical beacon chain head,
// fetching latest observed attestations, and more.
type BeaconServer struct {
	beaconDB            *db.BeaconDB
	ctx                 context.Context
	powChainService     powChainService
	chainService        chainService
	targetsFetcher      blockchain.TargetsFetcher
	operationService    operationService
	incomingAttestation chan *pbp2p.Attestation
	canonicalStateChan  chan *pbp2p.BeaconState
	chainStartChan      chan time.Time
}

// WaitForChainStart queries the logs of the Deposit Contract in order to verify the beacon chain
// has started its runtime and validators begin their responsibilities. If it has not, it then
// subscribes to an event stream triggered by the powchain service whenever the ChainStart log does
// occur in the Deposit Contract on ETH 1.0.
func (bs *BeaconServer) WaitForChainStart(req *ptypes.Empty, stream pb.BeaconService_WaitForChainStartServer) error {
	ok, err := bs.powChainService.HasChainStartLogOccurred()
	if err != nil {
		return fmt.Errorf("could not determine if ChainStart log has occurred: %v", err)
	}

	genesisTime, err := bs.powChainService.ETH2GenesisTime()
	if err != nil {
		return fmt.Errorf("could not determine chainstart time %v", err)
	}

	if ok {
		res := &pb.ChainStartResponse{
			Started:     true,
			GenesisTime: genesisTime,
		}
		return stream.Send(res)
	}

	sub := bs.chainService.StateInitializedFeed().Subscribe(bs.chainStartChan)
	defer sub.Unsubscribe()
	for {
		select {
		case chainStartTime := <-bs.chainStartChan:
			log.Info("Sending ChainStart log and genesis time to connected validator clients")
			res := &pb.ChainStartResponse{
				Started:     true,
				GenesisTime: uint64(chainStartTime.Unix()),
			}
			return stream.Send(res)
		case <-sub.Err():
			return errors.New("subscriber closed, exiting goroutine")
		case <-bs.ctx.Done():
			return errors.New("rpc context closed, exiting goroutine")
		}
	}
}

// CanonicalHead of the current beacon chain. This method is requested on-demand
// by a validator when it is their time to propose or attest.
func (bs *BeaconServer) CanonicalHead(ctx context.Context, req *ptypes.Empty) (*pbp2p.BeaconBlock, error) {
	block, err := bs.beaconDB.ChainHead()
	if err != nil {
		return nil, fmt.Errorf("could not get canonical head block: %v", err)
	}
	return block, nil
}

// BlockTree returns the current tree of saved blocks and their votes starting from the justified state.
func (bs *BeaconServer) BlockTree(ctx context.Context, _ *ptypes.Empty) (*pb.BlockTreeResponse, error) {
	justifiedState, err := bs.beaconDB.JustifiedState()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve justified state: %v", err)
	}
	attestationTargets, err := bs.targetsFetcher.AttestationTargets(justifiedState)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve attestation target: %v", err)
	}
	justifiedBlock, err := bs.beaconDB.JustifiedBlock()
	if err != nil {
		return nil, err
	}
	highestSlot := bs.beaconDB.HighestBlockSlot()
	fullBlockTree := []*pbp2p.BeaconBlock{}
	for i := justifiedBlock.Slot + 1; i < highestSlot; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		nextLayer, err := bs.beaconDB.BlocksBySlot(ctx, i)
		if err != nil {
			return nil, err
		}
		fullBlockTree = append(fullBlockTree, nextLayer...)
	}
	tree := []*pb.BlockTreeResponse_TreeNode{}
	for _, kid := range fullBlockTree {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		participatedVotes, err := blockchain.VoteCount(kid, justifiedState, attestationTargets, bs.beaconDB)
		if err != nil {
			return nil, err
		}
		blockRoot, err := blockutil.BlockSigningRoot(kid)
		if err != nil {
			return nil, err
		}
		tree = append(tree, &pb.BlockTreeResponse_TreeNode{
			BlockRoot:         blockRoot[:],
			Block:             kid,
			ParticipatedVotes: uint64(participatedVotes),
		})
	}
	return &pb.BlockTreeResponse{
		Tree: tree,
	}, nil
}

// eth1Data is a mechanism used by block proposers vote on a recent Ethereum 1.0 block hash and an
// associated deposit root found in the Ethereum 1.0 deposit contract. When consensus is formed,
// state.latest_eth1_data is updated, and validator deposits up to this root can be processed.
// The deposit root can be calculated by calling the get_deposit_root() function of
// the deposit contract using the post-state of the block hash.
func (bs *BeaconServer) eth1Data(ctx context.Context) (*pbp2p.Eth1Data, error) {
	beaconState, err := bs.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch beacon state: %v", err)
	}
	currentHeight := bs.powChainService.LatestBlockHeight()
	eth1FollowDistance := int64(params.BeaconConfig().Eth1FollowDistance)
	stateLatestEth1Hash := bytesutil.ToBytes32(beaconState.LatestEth1Data.DepositRoot)
	if stateLatestEth1Hash == [32]byte{} {
		return bs.defaultEth1DataResponse(ctx, currentHeight, eth1FollowDistance)
	}
	// Fetch the height of the block pointed to by the beacon state's latest_eth1_data.block_hash
	// in the canonical eth1.0 chain.
	_, stateLatestEth1Height, err := bs.powChainService.BlockExists(ctx, stateLatestEth1Hash)
	if err != nil {
		return nil, fmt.Errorf("could not verify block with hash exists in Eth1 chain: %#x: %v", stateLatestEth1Hash, err)
	}
	dataVotes := []*pbp2p.Eth1Data{}
	votesMap := helpers.EmptyVoteHierarchyMap()
	depositCount, depositRootAtHeight := bs.beaconDB.DepositsNumberAndRootAtHeight(ctx, currentHeight)
	for _, vote := range beaconState.Eth1DataVotes {
		validVote, blockHeight, err := bs.validateVote(ctx, currentHeight, depositCount, depositRootAtHeight, eth1FollowDistance, stateLatestEth1Height, vote)
		if err != nil {
			return nil, err
		}
		if validVote {
			dataVotes = append(dataVotes, vote)
			votesMap, err = helpers.CountVote(votesMap, vote, blockHeight)
			if err != nil {
				return nil, err
			}
		}
	}
	if len(dataVotes) == 0 {
		return bs.defaultEth1DataResponse(ctx, currentHeight, eth1FollowDistance)
	}
	return votesMap.BestVote, nil

}

func (bs *BeaconServer) validateVote(ctx context.Context, currentHeight *big.Int, depositCount uint64, depositRootAtHeight [32]byte, eth1FollowDistance int64, stateLatestEth1Height *big.Int, vote *pbp2p.Eth1Data) (bool, *big.Int, error) {
	if ctx.Err() != nil {
		return false, nil, ctx.Err()
	}
	eth1Hash := bytesutil.ToBytes32(vote.BlockHash)
	// Verify the block from the vote's block hash exists in the eth1.0 chain and fetch its height.
	blockExists, blockHeight, err := bs.powChainService.BlockExists(ctx, eth1Hash)
	if err != nil {
		log.WithError(err).WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(eth1Hash[:]))).
			Warn("Could not verify block with hash in ETH1 chain")
		return false, nil, nil
	}
	if !blockExists {
		return false, nil, nil
	}
	isBehindFollowDistance := big.NewInt(0).Sub(currentHeight, big.NewInt(eth1FollowDistance)).Cmp(blockHeight) >= 0
	isAheadStateLatestEth1Data := blockHeight.Cmp(stateLatestEth1Height) == 1
	correctDepositCount := depositCount == vote.DepositCount
	correctDepositRoot := bytes.Equal(vote.DepositRoot, depositRootAtHeight[:])
	return blockExists && isBehindFollowDistance && isAheadStateLatestEth1Data && correctDepositCount && correctDepositRoot, blockHeight, nil
}

// BlockTreeBySlots returns the current tree of saved blocks and their votes starting from the justified state.
func (bs *BeaconServer) BlockTreeBySlots(ctx context.Context, req *pb.TreeBlockSlotRequest) (*pb.BlockTreeResponse, error) {
	justifiedState, err := bs.beaconDB.JustifiedState()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve justified state: %v", err)
	}
	attestationTargets, err := bs.targetsFetcher.AttestationTargets(justifiedState)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve attestation target: %v", err)
	}
	justifiedBlock, err := bs.beaconDB.JustifiedBlock()
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New("argument 'TreeBlockSlotRequest' cannot be nil")
	}
	if !(req.SlotFrom <= req.SlotTo) {
		return nil, fmt.Errorf("upper limit (%d) of slot range cannot be lower than the lower limit (%d)", req.SlotTo, req.SlotFrom)
	}
	highestSlot := bs.beaconDB.HighestBlockSlot()
	fullBlockTree := []*pbp2p.BeaconBlock{}
	for i := justifiedBlock.Slot + 1; i < highestSlot; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if i >= req.SlotFrom && i <= req.SlotTo {
			nextLayer, err := bs.beaconDB.BlocksBySlot(ctx, i)
			if err != nil {
				return nil, err
			}
			if nextLayer != nil {
				fullBlockTree = append(fullBlockTree, nextLayer...)
			}
		}
		if i > req.SlotTo {
			break
		}
	}
	tree := []*pb.BlockTreeResponse_TreeNode{}
	for _, kid := range fullBlockTree {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		participatedVotes, err := blockchain.VoteCount(kid, justifiedState, attestationTargets, bs.beaconDB)
		if err != nil {
			return nil, err
		}
		blockRoot, err := blockutil.BlockSigningRoot(kid)
		if err != nil {
			return nil, err
		}
		hState, err := bs.beaconDB.HistoricalStateFromSlot(ctx, kid.Slot, blockRoot)
		if err != nil {
			return nil, err
		}
		if kid.Slot >= req.SlotFrom && kid.Slot <= req.SlotTo {
			activeValidatorIndices, err := helpers.ActiveValidatorIndices(hState, helpers.CurrentEpoch(hState))
			if err != nil {
				return nil, err
			}

			totalVotes, err := helpers.TotalBalance(hState, activeValidatorIndices)
			if err != nil {
				return nil, err
			}

			tree = append(tree, &pb.BlockTreeResponse_TreeNode{
				BlockRoot:         blockRoot[:],
				Block:             kid,
				ParticipatedVotes: uint64(participatedVotes),
				TotalVotes:        uint64(totalVotes),
			})
		}
	}
	return &pb.BlockTreeResponse{
		Tree: tree,
	}, nil
}

func (bs *BeaconServer) defaultEth1DataResponse(ctx context.Context, currentHeight *big.Int, eth1FollowDistance int64) (*pbp2p.Eth1Data, error) {
	ancestorHeight := big.NewInt(0).Sub(currentHeight, big.NewInt(eth1FollowDistance))
	blockHash, err := bs.powChainService.BlockHashByHeight(ctx, ancestorHeight)
	if err != nil {
		return nil, fmt.Errorf("could not fetch ETH1_FOLLOW_DISTANCE ancestor: %v", err)
	}
	// Fetch all historical deposits up to an ancestor height.
	depositsTillHeight, depositRoot := bs.beaconDB.DepositsNumberAndRootAtHeight(ctx, ancestorHeight)
	if depositsTillHeight == 0 {
		return nil, errors.New("could not fetch ETH1_FOLLOW_DISTANCE deposits")
	}
	return &pbp2p.Eth1Data{
		DepositRoot: depositRoot[:],
		BlockHash:   blockHash[:],
	}, nil
}

func constructMerkleProof(trie *trieutil.MerkleTrie, deposit *pbp2p.Deposit) (*pbp2p.Deposit, error) {
	proof, err := trie.MerkleProof(int(deposit.Index))
	if err != nil {
		return nil, fmt.Errorf(
			"could not generate merkle proof for deposit at index %d: %v",
			deposit.Index,
			err,
		)
	}
	// For every deposit, we construct a Merkle proof using the powchain service's
	// in-memory deposits trie, which is updated only once the state's LatestETH1Data
	// property changes during a state transition after a voting period.
	deposit.Proof = proof
	return deposit, nil
}

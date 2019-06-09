package rpc

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
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
		blockRoot, err := hashutil.HashBeaconBlock(kid)
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

// TODO(#2307): Refactor for v0.6.
// nolint
func (bs *BeaconServer) defaultDataResponse(ctx context.Context, currentHeight *big.Int, eth1FollowDistance int64) (*pbp2p.Eth1Data, error) {
	ancestorHeight := big.NewInt(0).Sub(currentHeight, big.NewInt(eth1FollowDistance))
	blockHash, err := bs.powChainService.BlockHashByHeight(ctx, ancestorHeight)
	if err != nil {
		return nil, fmt.Errorf("could not fetch ETH1_FOLLOW_DISTANCE ancestor: %v", err)
	}
	// Fetch all historical deposits up to an ancestor height.
	allDeposits := bs.beaconDB.AllDeposits(ctx, ancestorHeight)
	depositHashes := [][]byte{}
	// If there are less than or equal to len(ChainStartDeposits) historical deposits, then we just fetch the default
	// deposit root obtained from constructing the Merkle trie with the ChainStart deposits.
	chainStartDeposits, err := bs.powChainService.ChainStartDepositHashes()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve chainstart deposit hashes %v", err)
	}
	if len(allDeposits) <= len(chainStartDeposits) {
		depositHashes = chainStartDeposits
	} else {
		for i := range allDeposits {
			hash, err := hashutil.DepositHash(allDeposits[i].Data)
			if err != nil {
				return nil, err
			}
			depositHashes = append(depositHashes, hash[:])
		}
	}
	depositTrie, err := trieutil.GenerateTrieFromItems(depositHashes, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		return nil, fmt.Errorf("could not generate historical deposit trie from deposits: %v", err)
	}
	depositRoot := depositTrie.Root()
	return &pbp2p.Eth1Data{
		DepositRoot: depositRoot[:],
		BlockRoot:   blockHash[:],
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

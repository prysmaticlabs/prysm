package rpc

import (
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
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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

// LatestAttestation streams the latest processed attestations to the rpc clients.
func (bs *BeaconServer) LatestAttestation(req *ptypes.Empty, stream pb.BeaconService_LatestAttestationServer) error {
	sub := bs.operationService.IncomingAttFeed().Subscribe(bs.incomingAttestation)
	defer sub.Unsubscribe()
	for {
		select {
		case attestation := <-bs.incomingAttestation:
			log.Info("Sending attestation to RPC clients")
			if err := stream.Send(attestation); err != nil {
				return err
			}
		case <-sub.Err():
			log.Debug("Subscriber closed, exiting goroutine")
			return nil
		case <-bs.ctx.Done():
			log.Debug("RPC context closed, exiting goroutine")
			return nil
		}
	}
}

// DomainData fetches the current domain version information from the beacon state.
func (bs *BeaconServer) DomainData(ctx context.Context, request *pb.DomainRequest) (*pb.DomainResponse, error) {
	state, err := bs.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve beacon state: %v", err)
	}
	dv := helpers.DomainVersion(state, request.Epoch, params.BeaconConfig().DomainRandao)
	return &pb.DomainResponse{
		SignatureDomain: dv,
	}, nil

}

// Eth1Data is a mechanism used by block proposers vote on a recent Ethereum 1.0 block hash and an
// associated deposit root found in the Ethereum 1.0 deposit contract. When consensus is formed,
// state.latest_eth1_data is updated, and validator deposits up to this root can be processed.
// The deposit root can be calculated by calling the get_deposit_root() function of
// the deposit contract using the post-state of the block hash.
//
// TODO(#2307): Refactor for v0.6.
func (bs *BeaconServer) Eth1Data(ctx context.Context, _ *ptypes.Empty) (*pb.Eth1DataResponse, error) {
	return nil, nil
}

// PendingDeposits returns a list of pending deposits that are ready for
// inclusion in the next beacon block.
func (bs *BeaconServer) PendingDeposits(ctx context.Context, _ *ptypes.Empty) (*pb.PendingDepositsResponse, error) {
	bNum := bs.powChainService.LatestBlockHeight()
	if bNum == nil {
		return nil, errors.New("latest PoW block number is unknown")
	}
	// Only request deposits that have passed the ETH1 follow distance window.
	bNum = bNum.Sub(bNum, big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance)))
	allDeps := bs.beaconDB.AllDeposits(ctx, bNum)
	if len(allDeps) == 0 {
		return &pb.PendingDepositsResponse{PendingDeposits: nil}, nil
	}

	// Need to fetch if the deposits up to the state's latest eth 1 data matches
	// the number of all deposits in this RPC call. If not, then we return nil.
	beaconState, err := bs.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch beacon state: %v", err)
	}
	h := bytesutil.ToBytes32(beaconState.LatestEth1Data.BlockRoot)
	_, latestEth1DataHeight, err := bs.powChainService.BlockExists(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("could not fetch eth1data height: %v", err)
	}
	// If the state's latest eth1 data's block hash has a height of 100, we fetch all the deposits up to height 100.
	// If this doesn't match the number of deposits stored in the cache, the generated trie will not be the same and
	// root will fail to verify. This can happen in a scenario where we perhaps have a deposit from height 101,
	// so we want to avoid any possible mismatches in these lengths.
	upToLatestEth1DataDeposits := bs.beaconDB.AllDeposits(ctx, latestEth1DataHeight)
	if len(upToLatestEth1DataDeposits) != len(allDeps) {
		return &pb.PendingDepositsResponse{PendingDeposits: nil}, nil
	}
	depositData := [][]byte{}
	for _, dep := range upToLatestEth1DataDeposits {
		depHash, err := hashutil.DepositHash(dep.Data)
		if err != nil {
			return nil, fmt.Errorf("coulf not hash deposit data %v", err)
		}
		depositData = append(depositData, depHash[:])
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(depositData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		return nil, fmt.Errorf("could not generate historical deposit trie from deposits: %v", err)
	}

	allPendingDeps := bs.beaconDB.PendingDeposits(ctx, bNum)

	// Deposits need to be received in order of merkle index root, so this has to make sure
	// deposits are sorted from lowest to highest.
	var pendingDeps []*pbp2p.Deposit
	for _, dep := range allPendingDeps {
		if dep.Index >= beaconState.DepositIndex {
			pendingDeps = append(pendingDeps, dep)
		}
	}

	for i := range pendingDeps {
		// Don't construct merkle proof if the number of deposits is more than max allowed in block.
		if uint64(i) == params.BeaconConfig().MaxDeposits {
			break
		}
		pendingDeps[i], err = constructMerkleProof(depositTrie, pendingDeps[i])
		if err != nil {
			return nil, err
		}
	}
	// Limit the return of pending deposits to not be more than max deposits allowed in block.
	var pendingDeposits []*pbp2p.Deposit
	for i := 0; i < len(pendingDeps) && i < int(params.BeaconConfig().MaxDeposits); i++ {
		pendingDeposits = append(pendingDeposits, pendingDeps[i])
	}
	return &pb.PendingDepositsResponse{PendingDeposits: pendingDeposits}, nil
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
func (bs *BeaconServer) defaultDataResponse(ctx context.Context, currentHeight *big.Int, eth1FollowDistance int64) (*pb.Eth1DataResponse, error) {
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
	return &pb.Eth1DataResponse{
		Eth1Data: &pbp2p.Eth1Data{
			DepositRoot: depositRoot[:],
			BlockRoot:   blockHash[:],
		},
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

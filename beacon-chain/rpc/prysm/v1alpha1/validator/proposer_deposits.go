package validator

import (
	"bytes"
	"context"
	"math/big"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/trie"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (vs *Server) packDepositsAndAttestations(ctx context.Context, head state.BeaconState, eth1Data *ethpb.Eth1Data) ([]*ethpb.Deposit, []*ethpb.Attestation, error) {
	if features.Get().EnableGetBlockOptimizations {
		deposits, atts, err := vs.optimizedPackDepositsAndAttestations(ctx, head, eth1Data)
		if err != nil {
			return nil, nil, err
		}
		return deposits, atts, nil
	}

	// Pack ETH1 deposits which have not been included in the beacon chain.
	deposits, err := vs.deposits(ctx, head, eth1Data)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "Could not get ETH1 deposits: %v", err)
	}

	// Pack aggregated attestations which have not been included in the beacon chain.
	atts, err := vs.packAttestations(ctx, head)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "Could not get attestations to pack into block: %v", err)
	}

	return deposits, atts, nil
}

func (vs *Server) optimizedPackDepositsAndAttestations(ctx context.Context, head state.BeaconState, eth1Data *ethpb.Eth1Data) ([]*ethpb.Deposit, []*ethpb.Attestation, error) {
	eg, egctx := errgroup.WithContext(ctx)
	var deposits []*ethpb.Deposit
	var atts []*ethpb.Attestation

	eg.Go(func() error {
		// Pack ETH1 deposits which have not been included in the beacon chain.
		localDeposits, err := vs.deposits(egctx, head, eth1Data)
		if err != nil {
			return status.Errorf(codes.Internal, "Could not get ETH1 deposits: %v", err)
		}
		// if the original context is cancelled, then cancel this routine too
		select {
		case <-egctx.Done():
			return egctx.Err()
		default:
		}
		deposits = localDeposits
		return nil
	})

	eg.Go(func() error {
		// Pack aggregated attestations which have not been included in the beacon chain.
		localAtts, err := vs.packAttestations(egctx, head)
		if err != nil {
			return status.Errorf(codes.Internal, "Could not get attestations to pack into block: %v", err)
		}
		// if the original context is cancelled, then cancel this routine too
		select {
		case <-egctx.Done():
			return egctx.Err()
		default:
		}
		atts = localAtts
		return nil
	})

	return deposits, atts, eg.Wait()
}

// deposits returns a list of pending deposits that are ready for inclusion in the next beacon
// block. Determining deposits depends on the current eth1data vote for the block and whether or not
// this eth1data has enough support to be considered for deposits inclusion. If current vote has
// enough support, then use that vote for basis of determining deposits, otherwise use current state
// eth1data.
func (vs *Server) deposits(
	ctx context.Context,
	beaconState state.BeaconState,
	currentVote *ethpb.Eth1Data,
) ([]*ethpb.Deposit, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.deposits")
	defer span.End()

	if vs.MockEth1Votes {
		return []*ethpb.Deposit{}, nil
	}

	if !vs.Eth1InfoFetcher.IsConnectedToETH1() {
		log.Warn("not connected to eth1 node, skip pending deposit insertion")
		return []*ethpb.Deposit{}, nil
	}
	// Need to fetch if the deposits up to the state's latest eth1 data matches
	// the number of all deposits in this RPC call. If not, then we return nil.
	canonicalEth1Data, canonicalEth1DataHeight, err := vs.canonicalEth1Data(ctx, beaconState, currentVote)
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
		log.Debug("no pending deposits for inclusion in block")
		return []*ethpb.Deposit{}, nil
	}

	depositTrie, err := vs.depositTrie(ctx, canonicalEth1Data, canonicalEth1DataHeight)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve deposit trie")
	}

	// Deposits need to be received in order of merkle index root, so this has to make sure
	// deposits are sorted from lowest to highest.
	var pendingDeps []*ethpb.DepositContainer
	for _, dep := range allPendingContainers {
		if uint64(dep.Index) >= beaconState.Eth1DepositIndex() && uint64(dep.Index) < canonicalEth1Data.DepositCount {
			pendingDeps = append(pendingDeps, dep)
		}
		// Don't try to pack more than the max allowed in a block
		if uint64(len(pendingDeps)) == params.BeaconConfig().MaxDeposits {
			break
		}
	}

	for i := range pendingDeps {
		pendingDeps[i].Deposit, err = constructMerkleProof(depositTrie, int(pendingDeps[i].Index), pendingDeps[i].Deposit)
		if err != nil {
			return nil, err
		}
	}

	var pendingDeposits []*ethpb.Deposit
	for i := uint64(0); i < uint64(len(pendingDeps)); i++ {
		pendingDeposits = append(pendingDeposits, pendingDeps[i].Deposit)
	}
	return pendingDeposits, nil
}

func (vs *Server) depositTrie(ctx context.Context, canonicalEth1Data *ethpb.Eth1Data, canonicalEth1DataHeight *big.Int) (*trie.SparseMerkleTrie, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.depositTrie")
	defer span.End()

	var depositTrie *trie.SparseMerkleTrie

	finalizedDeposits := vs.DepositFetcher.FinalizedDeposits(ctx)
	depositTrie = finalizedDeposits.Deposits
	upToEth1DataDeposits := vs.DepositFetcher.NonFinalizedDeposits(ctx, canonicalEth1DataHeight)
	insertIndex := finalizedDeposits.MerkleTrieIndex + 1

	for _, dep := range upToEth1DataDeposits {
		depHash, err := dep.Data.HashTreeRoot()
		if err != nil {
			return nil, errors.Wrap(err, "could not hash deposit data")
		}
		if err = depositTrie.Insert(depHash[:], int(insertIndex)); err != nil {
			return nil, err
		}
		insertIndex++
	}
	valid, err := validateDepositTrie(depositTrie, canonicalEth1Data)
	// Log a warning here, as the cached trie is invalid.
	if !valid {
		log.Warnf("Cached deposit trie is invalid, rebuilding it now: %v", err)
		return vs.rebuildDepositTrie(ctx, canonicalEth1Data, canonicalEth1DataHeight)
	}

	return depositTrie, nil
}

// rebuilds our deposit trie by recreating it from all processed deposits till
// specified eth1 block height.
func (vs *Server) rebuildDepositTrie(ctx context.Context, canonicalEth1Data *ethpb.Eth1Data, canonicalEth1DataHeight *big.Int) (*trie.SparseMerkleTrie, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.rebuildDepositTrie")
	defer span.End()

	deposits := vs.DepositFetcher.AllDeposits(ctx, canonicalEth1DataHeight)
	trieItems := make([][]byte, 0, len(deposits))
	for _, dep := range deposits {
		depHash, err := dep.Data.HashTreeRoot()
		if err != nil {
			return nil, errors.Wrap(err, "could not hash deposit data")
		}
		trieItems = append(trieItems, depHash[:])
	}
	depositTrie, err := trie.GenerateTrieFromItems(trieItems, params.BeaconConfig().DepositContractTreeDepth)
	if err != nil {
		return nil, err
	}

	valid, err := validateDepositTrie(depositTrie, canonicalEth1Data)
	// Log an error here, as even with rebuilding the trie, it is still invalid.
	if !valid {
		log.Errorf("Rebuilt deposit trie is invalid: %v", err)
	}
	return depositTrie, nil
}

// validate that the provided deposit trie matches up with the canonical eth1 data provided.
func validateDepositTrie(trie *trie.SparseMerkleTrie, canonicalEth1Data *ethpb.Eth1Data) (bool, error) {
	if trie.NumOfItems() != int(canonicalEth1Data.DepositCount) {
		return false, errors.Errorf("wanted the canonical count of %d but received %d", canonicalEth1Data.DepositCount, trie.NumOfItems())
	}
	rt := trie.HashTreeRoot()
	if !bytes.Equal(rt[:], canonicalEth1Data.DepositRoot) {
		return false, errors.Errorf("wanted the canonical deposit root of %#x but received %#x", canonicalEth1Data.DepositRoot, rt)
	}
	return true, nil
}

func constructMerkleProof(trie *trie.SparseMerkleTrie, index int, deposit *ethpb.Deposit) (*ethpb.Deposit, error) {
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

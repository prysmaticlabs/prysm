package rpc

import (
	"context"
	"fmt"
	"math/big"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	newBlockchain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	blockchain "github.com/prysmaticlabs/prysm/beacon-chain/deprecated-blockchain"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// ProposerServer defines a server implementation of the gRPC Proposer service,
// providing RPC endpoints for computing state transitions and state roots, proposing
// beacon blocks to a beacon node, and more.
type ProposerServer struct {
	beaconDB           db.Database
	chainService       interface{}
	powChainService    powChainService
	operationService   operationService
	canonicalStateChan chan *pbp2p.BeaconState
	depositCache       *depositcache.DepositCache
}

// RequestBlock is called by a proposer during its assigned slot to request a block to sign
// by passing in the slot and the signed randao reveal of the slot.
func (ps *ProposerServer) RequestBlock(ctx context.Context, req *pb.BlockRequest) (*ethpb.BeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.RequestBlock")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(req.Slot)))

	// Retrieve the parent block as the current head of the canonical chain
	var parent *ethpb.BeaconBlock
	var err error
	if d, isLegacyDB := ps.beaconDB.(*db.BeaconDB); isLegacyDB {
		parent, err = d.ChainHead()
		if err != nil {
			return nil, errors.Wrap(err, "could not get canonical head block")
		}
	} else {
		parent, err = ps.beaconDB.(*kv.Store).HeadBlock(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get canonical head block")
		}
		parent = ps.chainService.(newBlockchain.HeadRetriever).HeadBlock()
	}

	parentRoot, err := ssz.SigningRoot(parent)
	if err != nil {
		return nil, errors.Wrap(err, "could not get parent block signing root")
	}

	// Construct block body
	// Pack ETH1 deposits which have not been included in the beacon chain
	eth1Data, err := ps.eth1Data(ctx, req.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get ETH1 data")
	}

	// Pack ETH1 deposits which have not been included in the beacon chain.
	deposits, err := ps.deposits(ctx, eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not get eth1 deposits")
	}

	// Pack aggregated attestations which have not been included in the beacon chain.
	attestations, err := ps.attestations(ctx, req.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get pending attestations")
	}

	// Use zero hash as stub for state root to compute later.
	stateRoot := params.BeaconConfig().ZeroHash[:]

	emptySig := make([]byte, 96)

	blk := &ethpb.BeaconBlock{
		Slot:       req.Slot,
		ParentRoot: parentRoot[:],
		StateRoot:  stateRoot,
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:     eth1Data,
			Deposits:     deposits,
			Attestations: attestations,
			RandaoReveal: req.RandaoReveal,
			// TODO(2766): Implement rest of the retrievals for beacon block operations
			Transfers:         []*ethpb.Transfer{},
			ProposerSlashings: []*ethpb.ProposerSlashing{},
			AttesterSlashings: []*ethpb.AttesterSlashing{},
			VoluntaryExits:    []*ethpb.VoluntaryExit{},
			Graffiti:          []byte{},
		},
		Signature: emptySig,
	}

	// Compute state root with the newly constructed block.
	stateRoot, err = ps.computeStateRoot(ctx, blk)
	if err != nil {
		return nil, errors.Wrap(err, "could not get compute state root")
	}
	blk.StateRoot = stateRoot

	return blk, nil
}

// ProposeBlock is called by a proposer during its assigned slot to create a block in an attempt
// to get it processed by the beacon node as the canonical head.
func (ps *ProposerServer) ProposeBlock(ctx context.Context, blk *ethpb.BeaconBlock) (*pb.ProposeResponse, error) {
	// TODO(#78): To protect against blk not filling, this will be handled within the SSZ code codebase.
	// https://github.com/prysmaticlabs/go-ssz/issues/78
	blk.Body.Graffiti = make([]byte, 32)
	root, err := ssz.SigningRoot(blk)
	if err != nil {
		return nil, errors.Wrap(err, "could not tree hash block")
	}
	log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(root[:]))).Debugf(
		"Block proposal received via RPC")

	db, isLegacyDB := ps.beaconDB.(*db.BeaconDB)
	if srv, isLegacyService := ps.chainService.(*blockchain.ChainService); isLegacyService && isLegacyDB {
		beaconState, err := srv.ReceiveBlockDeprecated(ctx, blk)
		if err != nil {
			return nil, errors.Wrap(err, "could not process beacon block")
		}
		if err := db.UpdateChainHead(ctx, blk, beaconState); err != nil {
			return nil, errors.Wrap(err, "failed to update chain")
		}
		ps.chainService.(*blockchain.ChainService).UpdateCanonicalRoots(blk, root)
	} else {
		if err := ps.chainService.(*newBlockchain.ChainService).ReceiveBlock(ctx, blk); err != nil {
			return nil, errors.Wrap(err, "could not process beacon block")
		}
	}

	return &pb.ProposeResponse{BlockRoot: root[:]}, nil
}

// attestations retrieves aggregated attestations kept in the beacon node's operations pool which have
// not yet been included into the beacon chain. Proposers include these pending attestations in their
// proposed blocks when performing their responsibility. If desired, callers can choose to filter pending
// attestations which are ready for inclusion. That is, attestations that satisfy:
// attestation.slot + MIN_ATTESTATION_INCLUSION_DELAY <= state.slot.
func (ps *ProposerServer) attestations(ctx context.Context, expectedSlot uint64) ([]*ethpb.Attestation, error) {
	var beaconState *pbp2p.BeaconState
	var err error
	if _, isLegacyDB := ps.beaconDB.(*db.BeaconDB); isLegacyDB {
		beaconState, err = ps.beaconDB.HeadState(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not retrieve beacon state")
		}
	} else {
		beaconState = ps.chainService.(newBlockchain.HeadRetriever).HeadState()
	}

	atts, err := ps.operationService.AttestationPool(ctx, expectedSlot)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve pending attestations from operations service")
	}

	// advance slot, if it is behind
	if beaconState.Slot < expectedSlot {
		beaconState, err = state.ProcessSlots(ctx, beaconState, expectedSlot)
		if err != nil {
			return nil, err
		}
	}

	var attsReadyForInclusion []*ethpb.Attestation
	for _, att := range atts {
		slot, err := helpers.AttestationDataSlot(beaconState, att.Data)
		if err != nil {
			return nil, errors.Wrap(err, "could not get attestation slot")
		}
		if slot+params.BeaconConfig().MinAttestationInclusionDelay <= beaconState.Slot &&
			beaconState.Slot <= slot+params.BeaconConfig().SlotsPerEpoch {
			attsReadyForInclusion = append(attsReadyForInclusion, att)
		}
	}

	validAtts := make([]*ethpb.Attestation, 0, len(attsReadyForInclusion))
	for _, att := range attsReadyForInclusion {
		slot, err := helpers.AttestationDataSlot(beaconState, att.Data)
		if err != nil {
			return nil, errors.Wrap(err, "could not get attestation slot")
		}

		if _, err := blocks.ProcessAttestationNoVerify(beaconState, att); err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}

			log.WithError(err).WithFields(logrus.Fields{
				"slot":     slot,
				"headRoot": fmt.Sprintf("%#x", bytesutil.Trunc(att.Data.BeaconBlockRoot))}).Info(
				"Deleting failed pending attestation from DB")

			var hash [32]byte
			if _, isLegacyDB := ps.beaconDB.(*db.BeaconDB); isLegacyDB {
				hash, err = ssz.HashTreeRoot(att)
				if err != nil {
					return nil, err
				}
			} else {
				hash, err = ssz.HashTreeRoot(att.Data)
				if err != nil {
					return nil, err
				}
			}
			if err := ps.beaconDB.DeleteAttestation(ctx, hash); err != nil {
				return nil, errors.Wrap(err, "could not delete failed attestation")
			}
			continue
		}
		validAtts = append(validAtts, att)
	}

	return validAtts, nil
}

// eth1Data determines the appropriate eth1data for a block proposal. The algorithm for this method
// is as follows:
//  - Determine the timestamp for the start slot for the eth1 voting period.
//  - Determine the most recent eth1 block before that timestamp.
//  - Subtract that eth1block.number by ETH1_FOLLOW_DISTANCE.
//  - This is the eth1block to use for the block proposal.
func (ps *ProposerServer) eth1Data(ctx context.Context, slot uint64) (*ethpb.Eth1Data, error) {
	eth1VotingPeriodStartTime, _ := ps.powChainService.ETH2GenesisTime()
	eth1VotingPeriodStartTime += (slot - (slot % params.BeaconConfig().SlotsPerEth1VotingPeriod)) * params.BeaconConfig().SecondsPerSlot

	// Look up most recent block up to timestamp
	blockNumber, err := ps.powChainService.BlockNumberByTimestamp(ctx, eth1VotingPeriodStartTime)
	if err != nil {
		return nil, err
	}

	return ps.defaultEth1DataResponse(ctx, blockNumber)
}

// computeStateRoot computes the state root after a block has been processed through a state transition and
// returns it to the validator client.
func (ps *ProposerServer) computeStateRoot(ctx context.Context, block *ethpb.BeaconBlock) ([]byte, error) {
	beaconState, err := ps.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve beacon state")
	}

	s, err := state.ExecuteStateTransitionNoVerify(
		ctx,
		beaconState,
		block,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not execute state transition for state at slot %d", beaconState.Slot)
	}

	root, err := ssz.HashTreeRoot(s)
	if err != nil {
		return nil, errors.Wrap(err, "could not tree hash beacon state")
	}
	log.WithField("beaconStateRoot", fmt.Sprintf("%#x", root)).Debugf("Computed state hash")
	return root[:], nil
}

// deposits returns a list of pending deposits that are ready for inclusion in the next beacon
// block. Determining deposits depends on the current eth1data vote for the block and whether or not
// this eth1data has enough support to be considered for deposits inclusion. If current vote has
// enough support, then use that vote for basis of determining deposits, otherwise use current state
// eth1data.
func (ps *ProposerServer) deposits(ctx context.Context, currentVote *ethpb.Eth1Data) ([]*ethpb.Deposit, error) {
	// Need to fetch if the deposits up to the state's latest eth 1 data matches
	// the number of all deposits in this RPC call. If not, then we return nil.
	var beaconState *pbp2p.BeaconState
	var err error
	if _, isLegacyDB := ps.beaconDB.(*db.BeaconDB); isLegacyDB {
		beaconState, err = ps.beaconDB.HeadState(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not retrieve beacon state")
		}
	} else {
		beaconState = ps.chainService.(newBlockchain.HeadRetriever).HeadState()
	}

	canonicalEth1Data, latestEth1DataHeight, err := ps.canonicalEth1Data(ctx, beaconState, currentVote)
	if err != nil {
		return nil, err
	}

	_, genesisEth1Block := ps.powChainService.ETH2GenesisTime()
	if genesisEth1Block.Cmp(latestEth1DataHeight) == 0 {
		return []*ethpb.Deposit{}, nil
	}

	upToEth1DataDeposits := ps.depositCache.AllDeposits(ctx, latestEth1DataHeight)
	depositData := [][]byte{}
	for _, dep := range upToEth1DataDeposits {
		depHash, err := ssz.HashTreeRoot(dep.Data)
		if err != nil {
			return nil, errors.Wrap(err, "could not hash deposit data")
		}
		depositData = append(depositData, depHash[:])
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(depositData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		return nil, errors.Wrap(err, "could not generate historical deposit trie from deposits")
	}

	allPendingContainers := ps.depositCache.PendingContainers(ctx, latestEth1DataHeight)

	// Deposits need to be received in order of merkle index root, so this has to make sure
	// deposits are sorted from lowest to highest.
	var pendingDeps []*depositcache.DepositContainer
	for _, dep := range allPendingContainers {
		if uint64(dep.Index) >= beaconState.Eth1DepositIndex && uint64(dep.Index) < canonicalEth1Data.DepositCount {
			pendingDeps = append(pendingDeps, dep)
		}
	}

	for i := range pendingDeps {
		// Don't construct merkle proof if the number of deposits is more than max allowed in block.
		if uint64(i) == params.BeaconConfig().MaxDeposits {
			break
		}
		pendingDeps[i].Deposit, err = constructMerkleProof(depositTrie, pendingDeps[i].Index, pendingDeps[i].Deposit)
		if err != nil {
			return nil, err
		}
	}
	// Limit the return of pending deposits to not be more than max deposits allowed in block.
	var pendingDeposits []*ethpb.Deposit
	for i := 0; i < len(pendingDeps) && i < int(params.BeaconConfig().MaxDeposits); i++ {
		pendingDeposits = append(pendingDeposits, pendingDeps[i].Deposit)
	}
	return pendingDeposits, nil
}

// canonicalEth1Data determines the canonical eth1data and eth1 block height to use for determining deposits.
func (ps *ProposerServer) canonicalEth1Data(ctx context.Context, beaconState *pbp2p.BeaconState, currentVote *ethpb.Eth1Data) (*ethpb.Eth1Data, *big.Int, error) {
	var eth1BlockHash [32]byte

	// Add in current vote, to get accurate vote tally
	beaconState.Eth1DataVotes = append(beaconState.Eth1DataVotes, currentVote)
	hasSupport, err := blocks.Eth1DataHasEnoughSupport(beaconState, currentVote)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not determine if current eth1data vote has enough support")
	}
	var canonicalEth1Data *ethpb.Eth1Data
	if hasSupport {
		canonicalEth1Data = currentVote
		eth1BlockHash = bytesutil.ToBytes32(currentVote.BlockHash)
	} else {
		canonicalEth1Data = beaconState.Eth1Data
		eth1BlockHash = bytesutil.ToBytes32(beaconState.Eth1Data.BlockHash)
	}
	_, latestEth1DataHeight, err := ps.powChainService.BlockExists(ctx, eth1BlockHash)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not fetch eth1data height")
	}
	return canonicalEth1Data, latestEth1DataHeight, nil
}

// in case no vote for new eth1data vote considered best vote we
// default into returning the latest deposit root and the block
// hash of eth1 block hash that is FOLLOW_DISTANCE back from its
// latest block.
func (ps *ProposerServer) defaultEth1DataResponse(ctx context.Context, currentHeight *big.Int) (*ethpb.Eth1Data, error) {
	eth1FollowDistance := int64(params.BeaconConfig().Eth1FollowDistance)
	ancestorHeight := big.NewInt(0).Sub(currentHeight, big.NewInt(eth1FollowDistance))
	blockHash, err := ps.powChainService.BlockHashByHeight(ctx, ancestorHeight)
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch ETH1_FOLLOW_DISTANCE ancestor")
	}
	// Fetch all historical deposits up to an ancestor height.
	depositsTillHeight, depositRoot := ps.depositCache.DepositsNumberAndRootAtHeight(ctx, ancestorHeight)
	if depositsTillHeight == 0 {
		return ps.powChainService.ChainStartETH1Data(), nil
	}
	return &ethpb.Eth1Data{
		DepositRoot:  depositRoot[:],
		BlockHash:    blockHash[:],
		DepositCount: depositsTillHeight,
	}, nil
}

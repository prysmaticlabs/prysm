package evaluators

import (
	"bytes"
	"context"
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"google.golang.org/grpc"
)

// PeersConnect checks all beacon nodes and returns whether they are connected to each other as peers.
var PeersConnect = Evaluator{
	Name:       "peers_connect_epoch_%d",
	Policy:     onEpoch(0),
	Evaluation: peersConnect,
}

// FinishedSyncing returns whether the beacon node with the given rpc port has finished syncing.
var FinishedSyncing = Evaluator{
	Name:       "finished_syncing",
	Policy:     func(currentEpoch uint64) bool { return true },
	Evaluation: finishedSyncing,
}

// AllNodesHaveSameHead ensures all nodes have the same head epoch. Checks finality and justification as well.
// Not checking head block root as it may change irregularly for the validator connected nodes.
var AllNodesHaveSameHead = Evaluator{
	Name:       "all_nodes_have_same_head",
	Policy:     func(currentEpoch uint64) bool { return true },
	Evaluation: allNodesHaveSameHead,
}

func onEpoch(epoch uint64) func(uint64) bool {
	return func(currentEpoch uint64) bool {
		return currentEpoch == epoch
	}
}

func peersConnect(conns ...*grpc.ClientConn) error {
	if len(conns) == 1 {
		return nil
	}
	ctx := context.Background()
	for _, conn := range conns {
		nodeClient := eth.NewNodeClient(conn)
		peersResp, err := nodeClient.ListPeers(ctx, &ptypes.Empty{})
		if err != nil {
			return err
		}
		expectedPeers := len(conns) - 1
		if expectedPeers != len(peersResp.Peers) {
			return fmt.Errorf("unexpected amount of peers, expected %d, received %d", expectedPeers, len(peersResp.Peers))
		}
	}
	return nil
}

func finishedSyncing(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	syncNodeClient := eth.NewNodeClient(conn)
	syncStatus, err := syncNodeClient.GetSyncStatus(context.Background(), &ptypes.Empty{})
	if err != nil {
		return err
	}
	if syncStatus.Syncing {
		return errors.New("expected node to have completed sync")
	}
	return nil
}

func allNodesHaveSameHead(conns ...*grpc.ClientConn) error {
	headEpochs := make([]uint64, len(conns))
	justifiedRoots := make([][]byte, len(conns))
	prevJustifiedRoots := make([][]byte, len(conns))
	finalizedRoots := make([][]byte, len(conns))
	for i, conn := range conns {
		beaconClient := eth.NewBeaconChainClient(conn)
		chainHead, err := beaconClient.GetChainHead(context.Background(), &ptypes.Empty{})
		if err != nil {
			return err
		}
		headEpochs[i] = chainHead.HeadEpoch
		justifiedRoots[i] = chainHead.JustifiedBlockRoot
		prevJustifiedRoots[i] = chainHead.PreviousJustifiedBlockRoot
		finalizedRoots[i] = chainHead.FinalizedBlockRoot
		if err := conn.Close(); err != nil {
			return err
		}
	}

	for i, epoch := range headEpochs {
		if headEpochs[0] != epoch {
			return fmt.Errorf(
				"received conflicting head epochs on node %d, expected %d, received %d",
				i,
				headEpochs[0],
				epoch,
			)
		}
	}
	for i, root := range justifiedRoots {
		if !bytes.Equal(justifiedRoots[0], root) {
			return fmt.Errorf(
				"received conflicting justified block roots on node %d, expected %#x, received %#x",
				i,
				justifiedRoots[0],
				root,
			)
		}
	}
	for i, root := range prevJustifiedRoots {
		if !bytes.Equal(prevJustifiedRoots[0], root) {
			return fmt.Errorf(
				"received conflicting previous justified block roots on node %d, expected %#x, received %#x",
				i,
				prevJustifiedRoots[0],
				root,
			)
		}
	}
	for i, root := range finalizedRoots {
		if !bytes.Equal(finalizedRoots[0], root) {
			return fmt.Errorf(
				"received conflicting finalized epoch roots on node %d, expected %#x, received %#x",
				i,
				finalizedRoots[0],
				root,
			)
		}
	}

	return nil
}

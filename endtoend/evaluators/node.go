package evaluators

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"google.golang.org/grpc"
)

// PeersConnect checks all beacon nodes and returns whether they are connected to each other as peers.
func PeersConnect(beaconNodes []*BeaconNodeInfo) error {
	for _, bNode := range beaconNodes {
		response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/p2p", bNode.MonitorPort))
		if err != nil {
			return errors.Wrap(err, "failed to reach p2p metrics page")
		}
		dataInBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		if err := response.Body.Close(); err != nil {
			return err
		}

		// Subtracting by 2 here since the libp2p page has "3 peers" as text.
		// With a starting index before the "p", going two characters back should give us
		// the number we need.
		startIdx := strings.Index(string(dataInBytes), "peers") - 2
		if startIdx == -3 {
			return fmt.Errorf("could not find needed text in %s", dataInBytes)
		}
		peerCount, err := strconv.Atoi(string(dataInBytes)[startIdx : startIdx+1])
		if err != nil {
			return err
		}
		expectedPeers := uint64(len(beaconNodes) - 1)
		if expectedPeers != uint64(peerCount) {
			return fmt.Errorf("unexpected amount of peers, expected %d, received %d", expectedPeers, peerCount)
		}
	}
	return nil
}

// FinishedSyncing returns whether the beacon node with the given rpc port has finished syncing.
func FinishedSyncing(rpcPort uint64) error {
	syncConn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", rpcPort), grpc.WithInsecure())
	if err != nil {
		return errors.Wrap(err, "failed to dial: %v")
	}
	syncNodeClient := eth.NewNodeClient(syncConn)
	syncStatus, err := syncNodeClient.GetSyncStatus(context.Background(), &ptypes.Empty{})
	if err != nil {
		return err
	}
	if syncStatus.Syncing {
		return errors.New("expected node to have completed sync")
	}
	return nil
}

// AllChainsHaveSameHead connects to all RPC ports in the passed in array and ensures they have the same head epoch.
// Checks finality and justification as well.
// Not checking head block root as it may change irregularly for the validator connected nodes.
func AllChainsHaveSameHead(beaconNodes []*BeaconNodeInfo) error {
	headEpochs := make([]uint64, len(beaconNodes))
	justifiedRoots := make([][]byte, len(beaconNodes))
	prevJustifiedRoots := make([][]byte, len(beaconNodes))
	finalizedRoots := make([][]byte, len(beaconNodes))
	for i, bNode := range beaconNodes {
		conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", bNode.RPCPort), grpc.WithInsecure())
		if err != nil {
			return errors.Wrap(err, "Failed to dial")
		}
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

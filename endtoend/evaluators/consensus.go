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

func allChainsHaveSameHead(grpcPorts []uint64) error {
	chainHeadRoots := make([][]byte, len(grpcPorts))
	justifiedRoots := make([][]byte, len(grpcPorts))
	prevJustifiedRoots := make([][]byte, len(grpcPorts))
	finalizedRoots := make([][]byte, len(grpcPorts))
	for i, port := range grpcPorts {
		conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", port), grpc.WithInsecure())
		if err != nil {
			return errors.Wrap(err, "Failed to dial")
		}
		beaconClient := eth.NewBeaconChainClient(conn)
		chainHead, err := beaconClient.GetChainHead(context.Background(), &ptypes.Empty{})
		if err != nil {
			return err
		}
		chainHeadRoots[i] = chainHead.HeadBlockRoot
		justifiedRoots[i] = chainHead.JustifiedBlockRoot
		prevJustifiedRoots[i] = chainHead.PreviousJustifiedBlockRoot
		finalizedRoots[i] = chainHead.FinalizedBlockRoot
		if err := conn.Close(); err != nil {
			return err
		}
	}
	for _, root := range chainHeadRoots {
		if !bytes.Equal(chainHeadRoots[0], root) {
			return fmt.Errorf(
				"received conflicting chain head block roots, expected %#x, received %#x",
				chainHeadRoots[0],
				root,
				)
		}
	}
	for _, root := range justifiedRoots {
		if !bytes.Equal(justifiedRoots[0], root) {
			return fmt.Errorf(
				"received conflicting justified block roots, expected %#x, received %#x",
				justifiedRoots[0],
				root,
				)
		}
	}
	for _, root := range prevJustifiedRoots {
		if !bytes.Equal(prevJustifiedRoots[0], root) {
			return fmt.Errorf(
				"received conflicting previous justified block roots, expected %#x, received %#x",
				prevJustifiedRoots[0],
				root,
			)
		}
	}
	for _, root := range finalizedRoots {
		if !bytes.Equal(finalizedRoots[0], root) {
			return fmt.Errorf(
				"received conflicting finalized epoch roots, expected %#x, received %#x",
				finalizedRoots[0],
				root,
			)
		}
	}

	return nil
}
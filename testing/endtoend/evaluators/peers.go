package evaluators

import (
	"context"

	"github.com/go-errors/errors"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/endtoend/policies"
	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// PeersCheck performs a check on peer data to ensure that any connected peers
// are not publishing invalid data.
var PeersCheck = types.Evaluator{
	Name:       "peers_check_epoch_%d",
	Policy:     policies.AfterNthEpoch(0),
	Evaluation: peersTest,
}

func peersTest(conns ...*grpc.ClientConn) error {
	debugClient := eth.NewDebugClient(conns[0])

	peerResponses, err := debugClient.ListPeers(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}
	for _, res := range peerResponses.Responses {
		if res.ScoreInfo.GossipScore < 0 {
			return errors.Errorf("Gossip score for peer %s is %f and negative.", res.PeerId, res.ScoreInfo.GossipScore)
		}
		if res.ScoreInfo.BehaviourPenalty > 0 {
			return errors.Errorf("Behaviour penalty for peer %s is %f and larger than zero.", res.PeerId, res.ScoreInfo.BehaviourPenalty)
		}
		if res.ScoreInfo.BlockProviderScore < 0 {
			return errors.Errorf("Block provider score for peer %s is %f and negative.", res.PeerId, res.ScoreInfo.BlockProviderScore)
		}
		if res.ScoreInfo.OverallScore < 0 {
			return errors.Errorf("Overall score for peer %s is %f and negative.", res.PeerId, res.ScoreInfo.OverallScore)
		}
		if res.ScoreInfo.ValidationError != "" {
			return errors.Errorf("Peer %s has a validation error: %s", res.PeerId, res.ScoreInfo.ValidationError)
		}
		if res.PeerInfo != nil && res.PeerInfo.FaultCount > 0 {
			return errors.Errorf("Peer %s has a non zero fault count: %d", res.PeerId, res.PeerInfo.FaultCount)
		}
		for topic, snap := range res.ScoreInfo.TopicScores {
			if snap.InvalidMessageDeliveries > 0 {
				return errors.Errorf("Peer %s in Topic %s has sent invalid deliveries: %f", res.PeerId, topic, snap.InvalidMessageDeliveries)
			}
		}
	}
	return nil
}

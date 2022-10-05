package p2p

import (
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	knownAgentVersions = []string{
		"lighthouse",
		"nimbus",
		"prysm",
		"teku",
		"lodestar",
		"js-libp2p",
		"rust-libp2p",
	}
	p2pPeerCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "p2p_peer_count",
		Help: "The number of peers in a given state.",
	},
		[]string{"state"})
	connectedPeersCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "connected_libp2p_peers",
		Help: "Tracks the total number of connected libp2p peers by agent string",
	},
		[]string{"agent"},
	)
	avgScoreConnectedClients = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "connected_libp2p_peers_average_scores",
		Help: "Tracks the overall p2p scores of connected libp2p peers by agent string",
	},
		[]string{"agent"},
	)
	repeatPeerConnections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "p2p_repeat_attempts",
		Help: "The number of repeat attempts the connection handler is triggered for a peer.",
	})
	statusMessageMissing = promauto.NewCounter(prometheus.CounterOpts{
		Name: "p2p_status_message_missing",
		Help: "The number of attempts the connection handler rejects a peer for a missing status message.",
	})
	savedAttestationBroadcasts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "p2p_attestation_subnet_recovered_broadcasts",
		Help: "The number of attestations that were attempted to be broadcast with no peers on " +
			"the subnet. The beacon node increments this counter when the broadcast is blocked " +
			"until a subnet peer can be found.",
	})
	attestationBroadcastAttempts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "p2p_attestation_subnet_attempted_broadcasts",
		Help: "The number of attestations that were attempted to be broadcast.",
	})
	savedSyncCommitteeBroadcasts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "p2p_sync_committee_subnet_recovered_broadcasts",
		Help: "The number of sync committee messages that were attempted to be broadcast with no peers on " +
			"the subnet. The beacon node increments this counter when the broadcast is blocked " +
			"until a subnet peer can be found.",
	})
	syncCommitteeBroadcastAttempts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "p2p_sync_committee_subnet_attempted_broadcasts",
		Help: "The number of sync committee that were attempted to be broadcast.",
	})
)

func (s *Service) updateMetrics() {
	connectedPeers := s.peers.Connected()
	p2pPeerCount.WithLabelValues("Connected").Set(float64(len(connectedPeers)))
	p2pPeerCount.WithLabelValues("Disconnected").Set(float64(len(s.peers.Disconnected())))
	p2pPeerCount.WithLabelValues("Connecting").Set(float64(len(s.peers.Connecting())))
	p2pPeerCount.WithLabelValues("Disconnecting").Set(float64(len(s.peers.Disconnecting())))
	p2pPeerCount.WithLabelValues("Bad").Set(float64(len(s.peers.Bad())))

	store := s.Host().Peerstore()
	numConnectedPeersByClient := make(map[string]float64)
	peerScoresByClient := make(map[string][]float64)
	for i := 0; i < len(connectedPeers); i++ {
		p := connectedPeers[i]
		pid, err := peer.Decode(p.String())
		if err != nil {
			log.WithError(err).Debug("Could not decode peer string")
			continue
		}

		// Get the agent data.
		rawAgent, err := store.Get(pid, "AgentVersion")
		agent, ok := rawAgent.(string)
		if err != nil || !ok {
			agent = "unknown"
		}
		foundName := "unknown"
		for _, knownAgent := range knownAgentVersions {
			// If the agent string matches one of our known agents, we set
			// the value to our own, sanitized string.
			if strings.Contains(strings.ToLower(agent), knownAgent) {
				foundName = knownAgent
			}
		}
		numConnectedPeersByClient[foundName] += 1

		// Get peer scoring data.
		overallScore := s.peers.Scorers().Score(pid)
		peerScoresByClient[foundName] = append(peerScoresByClient[foundName], overallScore)
	}
	for agent, total := range numConnectedPeersByClient {
		connectedPeersCount.WithLabelValues(agent).Set(total)
	}
	for agent, scoringData := range peerScoresByClient {
		avgScore := average(scoringData)
		avgScoreConnectedClients.WithLabelValues(agent).Set(avgScore)
	}
}

func average(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	total := 0.0
	for _, v := range xs {
		total += v
	}
	return total / float64(len(xs))
}

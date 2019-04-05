package attestation

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	validatorLastVoteGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "validators_last_vote",
		Help: "Votes of validators, updated when there's a new attestation",
	}, []string{
		"validatorIndex",
	})
	totalAttestationSeen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "total_seen_attestations",
		Help: "Total number of attestations seen by the validators",
	})
)

func reportVoteMetrics(index uint64, block *pb.BeaconBlock) {
	e := params.BeaconConfig().GenesisEpoch
	validatorLastVoteGauge.WithLabelValues(
		"v" + strconv.Itoa(int(index))).Set(float64(block.Slot - e))
}

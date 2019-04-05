package db

import (
	"encoding/hex"
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

func reportVoteMetrics(state *pb.BeaconState) {
	s := params.BeaconConfig().GenesisSlot
	e := params.BeaconConfig().GenesisEpoch
	currentEpoch := state.Slot / params.BeaconConfig().SlotsPerEpoch
	// Validator balances
	for i, bal := range state.ValidatorBalances {
		validatorBalancesGauge.WithLabelValues(
			"0x" + hex.EncodeToString(state.ValidatorRegistry[i].Pubkey), // Validator
		).Set(float64(bal))
	}

}

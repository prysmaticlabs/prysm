package slashings

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	numPendingAttesterSlashingFailedSigVerify = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "pending_attester_slashing_fail_sig_verify_total",
			Help: "Times an pending attester slashing fails sig verification",
		},
	)
	numPendingAttesterSlashings = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "num_pending_attester_slashings",
			Help: "Number of pending attester slashings in the pool",
		},
	)
	numAttesterSlashingsIncluded = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "attester_slashings_included_total",
			Help: "Number of attester slashings included in blocks",
		},
	)
	attesterSlashingReattempts = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "attester_slashing_reattempts_total",
			Help: "Times an attester slashing for an already slashed validator is received",
		},
	)
	numPendingProposerSlashingFailedSigVerify = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "pending_proposer_slashing_fail_sig_verify_total",
			Help: "Times an pending proposer slashing fails sig verification",
		},
	)
	numPendingProposerSlashings = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "num_pending_proposer_slashings",
			Help: "Number of pending proposer slashings in the pool",
		},
	)
	numProposerSlashingsIncluded = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "proposer_slashings_included_total",
			Help: "Number of proposer slashings included in blocks",
		},
	)
	proposerSlashingReattempts = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "proposer_slashing_reattempts_total",
			Help: "Times a proposer slashing for an already slashed validator is received",
		},
	)
)

package slashingprotection

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Counts slashable block proposal attempts detected by local slashing protection.
	localSlashableProposalsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "slashable_validator_proposals_rejected_local_total",
			Help: "Counts block proposal attempts rejected by slashing protection.",
		},
	)
	// Counts slashable block proposal attempts detected by remote slashing protection.
	remoteSlashableProposalsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "slashable_validator_proposals_rejected_remote_total",
			Help: "Counts block proposal attempts rejected by slashing protection.",
		},
	)
	// Counts slashable attestation attempts detected by local slashing protection.
	localSlashableAttestationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "slashable_validator_attestations_rejected_local_total",
			Help: "Counts attestation attempts rejected by local slashing protection.",
		},
	)
	// Counts slashable attestation attempts detected by remote slashing protection.
	remoteSlashableAttestationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "slashable_validator_attestations_rejected_remote_total",
			Help: "Counts attestation attempts rejected by remote slashing protection.",
		},
	)
)

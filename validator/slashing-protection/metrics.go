package slashingprotection

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Counts slashable block proposal attempts detected by local slashing protection.
	LocalSlashableProposalsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "slashable_validator_proposals_rejected_local_total",
			Help: "Counts block proposal attempts rejected by slashing protection.",
		},
	)
	// Counts slashable block proposal attempts detected by remote slashing protection.
	RemoteSlashableProposalsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "slashable_validator_proposals_rejected_remote_total",
			Help: "Counts block proposal attempts rejected by slashing protection.",
		},
	)
	// Counts slashable attestation attempts detected by local slashing protection.
	LocalSlashableAttestationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "slashable_validator_attestations_rejected_local_total",
			Help: "Counts attestation attempts rejected by local slashing protection.",
		},
	)
	// Counts slashable attestation attempts detected by remote slashing protection.
	RemoteSlashableAttestationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "slashable_validator_attestations_rejected_remote_total",
			Help: "Counts attestation attempts rejected by remote slashing protection.",
		},
	)
)

package slashingprotection

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Counts slashable block proposal attempts detected by local slashing protection.
	localSlashableProposalsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "slashable_validator_proposals_rejected_total_local",
			Help: "Counts block proposal attempts rejected by slashing protection.",
		},
		[]string{
			"pubkey",
		},
	)
	// Counts slashable block proposal attempts detected by remote slashing protection.
	remoteSlashableProposalsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "slashable_validator_proposals_rejected_total_remote",
			Help: "Counts block proposal attempts rejected by slashing protection.",
		},
		[]string{
			"pubkey",
		},
	)
	// Counts slashable attestation attempts detected by local slashing protection.
	localSlashableAttestationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "slashable_validator_attestations_rejected_local_total",
			Help: "Counts attestation attempts rejected by local slashing protection.",
		},
		[]string{
			"pubkey",
		},
	)
	// Counts slashable attestation attempts detected by remote slashing protection.
	remoteSlashableAttestationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "slashable_validator_attestations_rejected_remote_total",
			Help: "Counts attestation attempts rejected by remote slashing protection.",
		},
		[]string{
			"pubkey",
		},
	)
)

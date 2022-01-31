package remote_web3signer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	totalSignRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_sign_requests",
		Help: "Total number of sign requests",
	})
	totalErroredResponses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "web3signer_total_errored_responses",
		Help: "Total number of errored responses when calling web3signer",
	})
	totalBlockSignRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_block_sign_requests",
		Help: "Total number of block sign requests",
	})
	totalAggregationSlotSignRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_aggregation_slot_requests",
		Help: "Total number of aggregation slot requests",
	})
	totalAggregateAndProofSignRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_aggregate_and_proof_sign_requests",
		Help: "Total number of aggregate and proof sign requests",
	})
	totalAttestationSignRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_attestation_sign_requests",
		Help: "Total number of attestation sign requests",
	})
	totalBlockV2SignRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_block_v2_sign_requests",
		Help: "Total number of block v2 sign requests",
	})
	totalRandaoRevealSignRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_randao_reveal_sign_requests",
		Help: "Total number of randao reveal sign requests",
	})
	totalVoluntaryExitSignRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_voluntary_exit_sign_requests",
		Help: "Total number of voluntary exit sign requests",
	})
	totalSyncCommitteeMessageSignRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_sync_committee_message_sign_requests",
		Help: "Total number of sync committee message sign requests",
	})
	totalSyncCommitteeSelectionProofSignRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_sync_committee_selection_proof_sign_requests",
		Help: "Total number of sync committee selection proof sign requests",
	})
	totalSyncCommitteeContributionAndProofSignRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_sync_committee_contribution_and_proof_sign_requests",
		Help: "Total number of sync committee contribution and proof sign requests",
	})
)

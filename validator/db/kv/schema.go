package kv

var (
	// Validator slashing protection from double proposals.
	historicProposalsBucket = []byte("proposal-history-bucket")
	// Validator slashing protection from double proposals.
	newhistoricProposalsBucket = []byte("proposal-history-bucket-interchange")
	// Validator slashing protection from slashable attestations.
	historicAttestationsBucket = []byte("attestation-history-bucket")
	// New Validator slashing protection from slashable attestations.
	newHistoricAttestationsBucket = []byte("attestation-history-bucket-interchange")
)

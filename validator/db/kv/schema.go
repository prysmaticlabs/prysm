package kv

var (
	// Genesis information bucket key.
	genesisInfoBucket = []byte("genesis-info-bucket")

	// Validator slashing protection from double proposals.
	historicProposalsBucket = []byte("proposal-history-bucket")
	// Validator slashing protection from double proposals.
	newhistoricProposalsBucket = []byte("proposal-history-bucket-interchange")
	// Validator slashing protection from slashable attestations.
	historicAttestationsBucket = []byte("attestation-history-bucket")
	// New Validator slashing protection from slashable attestations.
	newHistoricAttestationsBucket = []byte("attestation-history-bucket-interchange")

	// Lowest and highest signed proposals.
	lowestSignedProposalsBucket  = []byte("lowest-signed-proposals-bucket")
	highestSignedProposalsBucket = []byte("highest-signed-proposals-bucket")

	// Genesis validators root bucket key.
	genesisValidatorsRootKey = []byte("genesis-val-root")
)

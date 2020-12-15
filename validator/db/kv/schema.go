package kv

var (
	// Genesis information bucket key.
	genesisInfoBucket = []byte("genesis-info-bucket")

	// Validator slashing protection from double proposals.
	historicProposalsBucket    = []byte("proposal-history-bucket")
	newHistoricProposalsBucket = []byte("proposal-history-bucket-interchange")

	// Validator slashing protection from slashable attestations.
	historicAttestationsBucket    = []byte("attestation-history-bucket")
	newHistoricAttestationsBucket = []byte("attestation-history-bucket-interchange")

	// Buckets for lowest signed source and target epoch for individual validator.
	lowestSignedSourceBucket = []byte("lowest-signed-source-bucket")
	lowestSignedTargetBucket = []byte("lowest-signed-target-bucket")

	// Lowest and highest signed proposals.
	lowestSignedProposalsBucket  = []byte("lowest-signed-proposals-bucket")
	highestSignedProposalsBucket = []byte("highest-signed-proposals-bucket")

	// Slashable public keys bucket.
	slashablePublicKeysBucket = []byte("slashable-public-keys")

	// Genesis validators root bucket key.
	genesisValidatorsRootKey = []byte("genesis-val-root")

	// Migrations.
	migrationsBucket = []byte("migrations")
)

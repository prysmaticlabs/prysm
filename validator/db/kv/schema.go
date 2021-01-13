package kv

var (
	// Genesis information bucket key.
	genesisInfoBucket = []byte("genesis-info-bucket")

	// Validator slashing protection from double proposals.
	historicProposalsBucket    = []byte("proposal-history-bucket-interchange")
	historicAttestationsBucket = []byte("attestation-history-bucket-interchange")

	// Buckets for lowest signed Source and Target epoch for individual validator.
	lowestSignedSourceBucket = []byte("lowest-signed-Source-bucket")
	lowestSignedTargetBucket = []byte("lowest-signed-Target-bucket")

	// Lowest and highest signed proposals.
	lowestSignedProposalsBucket  = []byte("lowest-signed-proposals-bucket")
	highestSignedProposalsBucket = []byte("highest-signed-proposals-bucket")

	// Slashable public keys bucket.
	slashablePublicKeysBucket = []byte("slashable-public-keys")

	// Genesis validators root bucket key.
	genesisValidatorsRootKey = []byte("genesis-val-root")

	// Optimized slashing protection buckets and keys.
	pubKeysBucket                 = []byte("pubkeys-bucket")
	attestationSigningRootsBucket = []byte("att-signing-roots-bucket")
	attestationSourceEpochsBucket = []byte("att-Source-epochs-bucket")

	// Migrations
	migrationsBucket = []byte("migrations")
)

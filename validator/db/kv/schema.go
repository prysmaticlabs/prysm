package kv

var (
	// Validator slashing protection from double proposals.
	historicProposalsBucket = []byte("proposal-history-bucket")
	// Validator slashing protection from slashable attestations.
	historicAttestationsBucket = []byte("attestation-history-bucket")
	// Bucket for storing important information regarding the validator API
	// such as a password hash for API authentication.
	validatorAPIBucket = []byte("validator-api-bucket")
	// Bucket key for retrieving the hashed password used for
	// authentication to the validator API.
	apiHashedPasswordKey = []byte("hashed-password")
)

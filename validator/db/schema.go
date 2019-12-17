package db

import ()

var (
	// Protector
	historicProposalsBucket = []byte("historic-proposals-bucket")
	// In order to quickly detect surround and surrounded attestations we need to store
	// the min and max span for each validator for each epoch.
	// see https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
	validatorsMinMaxSpanBucket = []byte("validators-min-max-span-bucket")
)

package db

import (
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

var (
	// Slasher
	historicIndexedAttestationsBucket = []byte("historic-indexed-attestations-bucket")
	historicBlockHeadersBucket        = []byte("historic-block-headers-bucket")
	proposerSlashingBucket            = []byte("proposer-slashing-bucket")
	attesterSlashingBucket            = []byte("attester-slashing-bucket")
	indexedAttestationsIndicesBucket  = []byte("indexed-attestations-indices-bucket")
	validatorsPublicKeysBucket        = []byte("validators-public-keys-bucket")
	// In order to quickly detect surround and surrounded attestations we need to store
	// the min and max span for each validator for each epoch.
	// see https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
	validatorsMinMaxSpanBucket = []byte("validators-min-max-span-bucket")
)

func encodeEpochValidatorID(epoch uint64, validatorID uint64) []byte {
	return append(bytesutil.Bytes8(epoch), bytesutil.Bytes8(validatorID)...)
}

func encodeEpochValidatorIDSig(epoch uint64, validatorID uint64, sig []byte) []byte {
	return append(append(bytesutil.Bytes8(epoch), bytesutil.Bytes8(validatorID)...), sig...)
}

func encodeEpochSig(targetEpoch uint64, sig []byte) []byte {
	return append(bytesutil.Bytes8(targetEpoch), sig...)
}

func encodeStatusValidatorIDRoot(status SlashingStatus, validatorID uint64, root [32]byte) []byte {
	return append(append([]byte{byte(status)}, bytesutil.Bytes8(validatorID)...), root[:]...)
}

func encodeStatusValidatorID(status SlashingStatus, validatorID uint64) []byte {
	return append([]byte{byte(status)}, bytesutil.Bytes8(validatorID)...)
}

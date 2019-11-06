package db

import (
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

var (
	// Slasher
	historicIndexedAttestationsBucket = []byte("historic-indexed-attestations-bucket")
	historicBlockHeadersBucket        = []byte("historic-block-headers-bucket")
	indexedAttestationsIndicesBucket  = []byte("indexed-attestations-indices-bucket")
	validatorsPublicKeysBucket        = []byte("validators-public-keys-bucket")
	// in order to quickly detect surround and surrounded attestations we need to store
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

func encodeEpochSig(sourceEpoch uint64, targetEpoch uint64, sig []byte) []byte {
	st := append(bytesutil.Bytes8(sourceEpoch), bytesutil.Bytes8(targetEpoch)...)
	return append(st, sig...)
}

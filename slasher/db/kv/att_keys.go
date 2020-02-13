package kv

import (
	"bytes"
	"context"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

func unmarshalIdxAttKeys(ctx context.Context, enc []byte) ([][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.unmarshalCompressedIdxAttList")
	defer span.End()
	uint64Length := 8
	keyLength := params.BeaconConfig().BLSSignatureLength + uint64Length
	if len(enc)%keyLength != 0 {
		return nil, fmt.Errorf("data length in keys array: %d is not a multiple of keys length: %d ", len(enc), keyLength)
	}
	keys := make([][]byte, len(enc)/keyLength)
	for i := range keys {
		keys[i] = enc[i*keyLength : (i+1)*keyLength]
	}
	return keys, nil
}

func addAttKeyToEpochValIDList(ctx context.Context, idxAttestation *ethpb.IndexedAttestation, tx *bolt.Tx) error {
	bucket := tx.Bucket(epochValidatorIdxAttsBucket)
	idxAttKey := encodeEpochSig(idxAttestation.Data.Target.Epoch, idxAttestation.Signature)
	for _, valIdx := range idxAttestation.AttestingIndices {
		key := encodeEpochValidatorID(idxAttestation.Data.Target.Epoch, valIdx)
		enc := bucket.Get(key)
		if enc == nil {
			if err := bucket.Put(key, idxAttKey); err != nil {
				return errors.Wrap(err, "failed to save indexed attestation into historical bucket")
			}
		}
		keys, err := unmarshalIdxAttKeys(ctx, enc)
		if err != nil {
			return errors.Wrap(err, "failed to marshal")
		}
		for _, k := range keys {
			if bytes.Equal(k, idxAttKey) {
				return nil
			}
		}
		if err := bucket.Put(key, append(enc, idxAttKey...)); err != nil {
			return errors.Wrap(err, "failed to save indexed attestation into historical bucket")
		}
	}
	return nil
}

func removeAttKeyFromEpochValIDList(ctx context.Context, idxAttestation *ethpb.IndexedAttestation, tx *bolt.Tx) error {
	idxAttKey := encodeEpochSig(idxAttestation.Data.Target.Epoch, idxAttestation.Signature)
	bucket := tx.Bucket(epochValidatorIdxAttsBucket)

	for _, valIdx := range idxAttestation.AttestingIndices {
		key := encodeEpochValidatorID(idxAttestation.Data.Target.Epoch, valIdx)
		enc := bucket.Get(key)
		if enc == nil {
			continue
		}
		keys, err := unmarshalIdxAttKeys(ctx, enc)
		if err != nil {
			return errors.Wrap(err, "failed to marshal")
		}
		for i, k := range keys {
			if bytes.Equal(k, idxAttKey) {
				keys = append(keys[:i], keys[i+1:]...)
				if err := bucket.Put(key, bytes.Join(keys, []byte{})); err != nil {
					return errors.Wrap(err, "failed to delete indexed attestation from historical bucket")
				}
			}
		}
	}
	return nil
}

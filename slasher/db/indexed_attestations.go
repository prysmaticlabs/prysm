package db

import (
	"bytes"
	"reflect"
	"sort"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func createIndexedAttestation(enc []byte) (*ethpb.IndexedAttestation, error) {
	protoIdxAtt := &ethpb.IndexedAttestation{}
	err := proto.Unmarshal(enc, protoIdxAtt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoIdxAtt, nil
}

func createValidatorIDsToIndexedAttestationList(enc []byte) (*ethpb.ValidatorIDToIdxAttList, error) {
	protoIdxAtt := &ethpb.ValidatorIDToIdxAttList{}
	err := proto.Unmarshal(enc, protoIdxAtt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoIdxAtt, nil
}

// IndexedAttestation accepts a epoch and validator index and returns a list of
// indexed attestations.
// Returns nil if the indexed attestation does not exist.
func (db *Store) IndexedAttestation(epoch uint64, validatorID uint64) ([]*ethpb.IndexedAttestation, error) {
	var iAtt []*ethpb.IndexedAttestation
	key := bytesutil.Bytes8(epoch)
	err := db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(indexedAttestationsIndicesBucket)
		enc := bucket.Get(key)
		iList, err := createValidatorIDsToIndexedAttestationList(enc)
		if err != nil {
			return err
		}
		for _, a := range iList.IndicesList {
			i := sort.Search(len(a.Indices), func(i int) bool { return a.Indices[i] >= validatorID })
			if i < len(a.Indices) && a.Indices[i] == validatorID {
				iaBucket := tx.Bucket(historicIndexedAttestationsBucket)
				key := encodeEpochSig(epoch, a.Signature)
				enc = iaBucket.Get(key)
				if len(enc) == 0 {
					continue
				}
				iA, err := createIndexedAttestation(enc)
				if err != nil {
					return err
				}
				iAtt = append(iAtt, iA)
			}
		}
		return nil
	})

	return iAtt, err
}

// HasIndexedAttestation accepts an epoch and validator id and returns true if the indexed attestation exists.
func (db *Store) HasIndexedAttestation(epoch uint64, validatorID uint64) bool {
	key := bytesutil.Bytes8(epoch)
	var hasAttestation bool
	// #nosec G104
	_ = db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(indexedAttestationsIndicesBucket)
		enc := bucket.Get(key)
		iList, err := createValidatorIDsToIndexedAttestationList(enc)
		if err != nil {
			return err
		}
		for _, a := range iList.IndicesList {
			i := sort.Search(len(a.Indices), func(i int) bool { return a.Indices[i] >= validatorID })
			if i < len(a.Indices) && a.Indices[i] == validatorID {
				hasAttestation = true
				return nil
			}
		}
		hasAttestation = false
		return nil
	})

	return hasAttestation
}

// SaveIndexedAttestation accepts epoch and indexed attestation and writes it to disk.
func (db *Store) SaveIndexedAttestation(epoch uint64, idxAttestation *ethpb.IndexedAttestation) error {
	key := encodeEpochSig(epoch, idxAttestation.Signature)
	enc, err := proto.Marshal(idxAttestation)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}

	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		//if data is in db skip put and index functions
		val := bucket.Get(key)
		if val != nil {
			return nil
		}
		createIndexedAttestationIndicesFromData(epoch, idxAttestation, tx)
		if err := bucket.Put(key, enc); err != nil {
			return errors.Wrap(err, "failed to include the indexed attestation in the historic indexed attestation bucket")
		}

		return err
	})

	// prune history to max size every 10th epoch
	if epoch%10 == 0 {
		weakSubjectivityPeriod := uint64(54000)
		err = db.PruneHistory(epoch, weakSubjectivityPeriod)
	}
	return err
}

func createIndexedAttestationIndicesFromData(epoch uint64, idxAttestation *ethpb.IndexedAttestation, tx *bolt.Tx) error {
	indices := append(idxAttestation.CustodyBit_0Indices, idxAttestation.CustodyBit_1Indices...)
	dataRoot, err := ssz.Marshal(idxAttestation.Data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal indexed attestation data.")
	}
	protoIdxAtt := &ethpb.ValidatorIDToIdxAtt{
		Signature: idxAttestation.Signature,
		Indices:   indices,
		DataRoot:  dataRoot,
	}
	key := bytesutil.Bytes8(epoch)
	bucket := tx.Bucket(indexedAttestationsIndicesBucket)
	enc := bucket.Get(key)
	vIdxList, err := createValidatorIDsToIndexedAttestationList(enc)
	if err != nil {
		return errors.Wrap(err, "failed to decode value into ValidatorIDToIndexedAttestationList")
	}
	vIdxList.IndicesList = append(vIdxList.IndicesList, protoIdxAtt)
	enc, err = proto.Marshal(vIdxList)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}
	if err := bucket.Put(key, enc); err != nil {
		return errors.Wrap(err, "failed to include the indexed attestation in the historic indexed attestation bucket")
	}
	return nil
}

// DeleteIndexedAttestation deletes a indexed attestation using the slot and its root as keys in their respective buckets.
func (db *Store) DeleteIndexedAttestation(epoch uint64, idxAttestation *ethpb.IndexedAttestation) error {
	key := encodeEpochSig(epoch, idxAttestation.Signature)
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		enc := bucket.Get(key)
		if enc == nil {
			return nil
		}
		removeIndexedAttestationIndicesFromData(epoch, idxAttestation, tx)
		if err := bucket.Delete(key); err != nil {
			return errors.Wrap(err, "failed to delete the indexed attestation from historic indexed attestation bucket")
		}
		return nil
	})
}

func removeIndexedAttestationIndicesFromData(epoch uint64, idxAttestation *ethpb.IndexedAttestation, tx *bolt.Tx) error {
	indices := append(idxAttestation.CustodyBit_0Indices, idxAttestation.CustodyBit_1Indices...)
	dataRoot, err := ssz.Marshal(idxAttestation.Data)
	protoIdxAtt := &ethpb.ValidatorIDToIdxAtt{
		Signature: idxAttestation.Signature,
		Indices:   indices,
		DataRoot:  dataRoot,
	}
	key := bytesutil.Bytes8(epoch)
	bucket := tx.Bucket(indexedAttestationsIndicesBucket)
	enc := bucket.Get(key)
	vIdxList, err := createValidatorIDsToIndexedAttestationList(enc)
	if err != nil {
		return errors.Wrap(err, "failed to decode value into ValidatorIDToIndexedAttestationList")
	}
	for i, v := range vIdxList.IndicesList {
		if reflect.DeepEqual(v, protoIdxAtt) {
			copy(vIdxList.IndicesList[i:], vIdxList.IndicesList[i+1:])
			vIdxList.IndicesList[len(vIdxList.IndicesList)-1] = nil // or the zero value of T
			vIdxList.IndicesList = vIdxList.IndicesList[:len(vIdxList.IndicesList)-1]
			break
		}
	}
	enc, err = proto.Marshal(vIdxList)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}
	if err := bucket.Put(key, enc); err != nil {
		return errors.Wrap(err, "failed to include the indexed attestation in the historic indexed attestation bucket")
	}
	return nil
}

func (db *Store) pruneAttHistory(currentEpoch uint64, historySize uint64) error {
	pruneTill := int64(currentEpoch) - int64(historySize)
	if pruneTill <= 0 {
		return nil
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		max := bytesutil.Bytes8(uint64(pruneTill))
		for k, _ := c.First(); k != nil && bytes.Compare(k[:8], max) <= 0; k, _ = c.Next() {
			if err := bucket.Delete(k); err != nil {
				return errors.Wrap(err, "failed to delete the indexed attestation from historic indexed attestation bucket")
			}
		}
		idxBucket := tx.Bucket(indexedAttestationsIndicesBucket)
		c = tx.Bucket(indexedAttestationsIndicesBucket).Cursor()
		for k, _ := c.First(); k != nil && bytes.Compare(k[:8], max) <= 0; k, _ = c.Next() {
			if err := idxBucket.Delete(k); err != nil {
				return errors.Wrap(err, "failed to delete the indexed attestation from indexed attestation indexes bucket")
			}
		}
		return nil
	})
}

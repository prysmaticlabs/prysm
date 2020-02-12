package db

import (
	"bytes"
	"reflect"
	"sort"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func unmarshalIdxAtt(enc []byte) (*ethpb.IndexedAttestation, error) {
	protoIdxAtt := &ethpb.IndexedAttestation{}
	err := proto.Unmarshal(enc, protoIdxAtt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoded indexed attestation")
	}
	return protoIdxAtt, nil
}

func unmarshalCompressedIdxAttList(enc []byte) (*slashpb.CompressedIdxAttList, error) {
	protoIdxAtt := &slashpb.CompressedIdxAttList{}
	err := proto.Unmarshal(enc, protoIdxAtt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoIdxAtt, nil
}

// IdxAttsForTargetFromID accepts a epoch and validator index and returns a list of
// indexed attestations from that validator for the given target epoch.
// Returns nil if the indexed attestation does not exist.
func (db *Store) IdxAttsForTargetFromID(targetEpoch uint64, validatorID uint64) ([]*ethpb.IndexedAttestation, error) {
	var idxAtts []*ethpb.IndexedAttestation

	err := db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(compressedIdxAttsBucket)
		enc := bucket.Get(bytesutil.Bytes8(targetEpoch))
		if enc == nil {
			return nil
		}
		idToIdxAttsList, err := unmarshalCompressedIdxAttList(enc)
		if err != nil {
			return err
		}

		for _, idxAtt := range idToIdxAttsList.List {
			i := sort.Search(len(idxAtt.Indices), func(i int) bool {
				return idxAtt.Indices[i] >= validatorID
			})
			if i < len(idxAtt.Indices) && idxAtt.Indices[i] == validatorID {
				idxAttBucket := tx.Bucket(historicIndexedAttestationsBucket)
				key := encodeEpochSig(targetEpoch, idxAtt.Signature)
				enc = idxAttBucket.Get(key)
				if enc == nil {
					continue
				}
				att, err := unmarshalIdxAtt(enc)
				if err != nil {
					return err
				}
				idxAtts = append(idxAtts, att)
			}
		}
		return nil
	})

	return idxAtts, err
}

// IdxAttsForTarget accepts a target epoch and returns a list of
// indexed attestations.
// Returns nil if the indexed attestation does not exist with that target epoch.
func (db *Store) IdxAttsForTarget(targetEpoch uint64) ([]*ethpb.IndexedAttestation, error) {
	var idxAtts []*ethpb.IndexedAttestation
	key := bytesutil.Bytes8(targetEpoch)
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		for k, enc := c.Seek(key); k != nil && bytes.Equal(k[:8], key); k, _ = c.Next() {
			idxAtt, err := unmarshalIdxAtt(enc)
			if err != nil {
				return err
			}
			idxAtts = append(idxAtts, idxAtt)
		}
		return nil
	})
	return idxAtts, err
}

// LatestIndexedAttestationsTargetEpoch returns latest target epoch in db
// returns 0 if there is no indexed attestations in db.
func (db *Store) LatestIndexedAttestationsTargetEpoch() (uint64, error) {
	var lt uint64
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		k, _ := c.Last()
		if k == nil {
			return nil
		}
		lt = bytesutil.FromBytes8(k[:8])
		return nil
	})
	return lt, err
}

// LatestValidatorIdx returns latest validator id in db
// returns 0 if there is no validators in db.
func (db *Store) LatestValidatorIdx() (uint64, error) {
	var lt uint64
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(compressedIdxAttsBucket).Cursor()
		k, _ := c.Last()
		if k == nil {
			return nil
		}
		lt = bytesutil.FromBytes8(k[:8])
		return nil
	})
	return lt, err
}

// HasIndexedAttestation accepts an epoch and validator id and returns true if the indexed attestation exists.
func (db *Store) HasIndexedAttestation(targetEpoch uint64, validatorID uint64) (bool, error) {
	key := bytesutil.Bytes8(targetEpoch)
	var hasAttestation bool
	// #nosec G104
	err := db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(compressedIdxAttsBucket)
		enc := bucket.Get(key)
		if enc == nil {
			return nil
		}
		iList, err := unmarshalCompressedIdxAttList(enc)
		if err != nil {
			return err
		}
		for _, idxAtt := range iList.List {
			i := sort.Search(len(idxAtt.Indices), func(i int) bool {
				return idxAtt.Indices[i] >= validatorID
			})
			if i < len(idxAtt.Indices) && idxAtt.Indices[i] == validatorID {
				hasAttestation = true
				return nil
			}
		}
		return nil
	})

	return hasAttestation, err
}

// SaveIndexedAttestation accepts epoch and indexed attestation and writes it to disk.
func (db *Store) SaveIndexedAttestation(idxAttestation *ethpb.IndexedAttestation) error {
	key := encodeEpochSig(idxAttestation.Data.Target.Epoch, idxAttestation.Signature)
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
		if err := saveCompressedIdxAttToEpochList(idxAttestation, tx); err != nil {
			return errors.Wrap(err, "failed to save indices from indexed attestation")
		}
		if err := bucket.Put(key, enc); err != nil {
			return errors.Wrap(err, "failed to save indexed attestation into historical bucket")
		}

		return err
	})

	// Prune history to max size every PruneSlasherStoragePeriod epoch.
	if idxAttestation.Data.Source.Epoch%params.BeaconConfig().PruneSlasherStoragePeriod == 0 {
		wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
		if err = db.pruneAttHistory(idxAttestation.Data.Source.Epoch, wsPeriod); err != nil {
			return err
		}
	}
	return err
}

func saveCompressedIdxAttToEpochList(idxAttestation *ethpb.IndexedAttestation, tx *bolt.Tx) error {
	dataRoot, err := hashutil.HashProto(idxAttestation.Data)
	if err != nil {
		return errors.Wrap(err, "failed to hash indexed attestation data.")
	}
	protoIdxAtt := &slashpb.CompressedIdxAtt{
		Signature: idxAttestation.Signature,
		Indices:   idxAttestation.AttestingIndices,
		DataRoot:  dataRoot[:],
	}

	key := bytesutil.Bytes8(idxAttestation.Data.Target.Epoch)
	bucket := tx.Bucket(compressedIdxAttsBucket)
	enc := bucket.Get(key)
	compressedIdxAttList, err := unmarshalCompressedIdxAttList(enc)
	if err != nil {
		return errors.Wrap(err, "failed to decode value into CompressedIdxAtt")
	}
	compressedIdxAttList.List = append(compressedIdxAttList.List, protoIdxAtt)
	enc, err = proto.Marshal(compressedIdxAttList)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}
	if err := bucket.Put(key, enc); err != nil {
		return errors.Wrap(err, "failed to save indexed attestation into historical bucket")
	}
	return nil
}

// DeleteIndexedAttestation deletes a indexed attestation using the slot and its root as keys in their respective buckets.
func (db *Store) DeleteIndexedAttestation(idxAttestation *ethpb.IndexedAttestation) error {
	key := encodeEpochSig(idxAttestation.Data.Target.Epoch, idxAttestation.Signature)
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		enc := bucket.Get(key)
		if enc == nil {
			return nil
		}
		if err := removeIdxAttIndicesByEpochFromDB(idxAttestation, tx); err != nil {
			return err
		}
		if err := bucket.Delete(key); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				return errors.Wrapf(rollbackErr, "failed to rollback after %v", err)
			}
			return errors.Wrap(err, "failed to delete indexed attestation from historical bucket")
		}
		return nil
	})
}

func removeIdxAttIndicesByEpochFromDB(idxAttestation *ethpb.IndexedAttestation, tx *bolt.Tx) error {
	dataRoot, err := hashutil.HashProto(idxAttestation.Data)
	if err != nil {
		return err
	}
	protoIdxAtt := &slashpb.CompressedIdxAtt{
		Signature: idxAttestation.Signature,
		Indices:   idxAttestation.AttestingIndices,
		DataRoot:  dataRoot[:],
	}
	key := bytesutil.Bytes8(idxAttestation.Data.Target.Epoch)
	bucket := tx.Bucket(compressedIdxAttsBucket)
	enc := bucket.Get(key)
	if enc == nil {
		return errors.New("requested to delete data that is not present")
	}
	vIdxList, err := unmarshalCompressedIdxAttList(enc)
	if err != nil {
		return errors.Wrap(err, "failed to decode value into ValidatorIDToIndexedAttestationList")
	}
	for i, attIdx := range vIdxList.List {
		if reflect.DeepEqual(attIdx, protoIdxAtt) {
			copy(vIdxList.List[i:], vIdxList.List[i+1:])
			vIdxList.List[len(vIdxList.List)-1] = nil // or the zero value of T
			vIdxList.List = vIdxList.List[:len(vIdxList.List)-1]
			break
		}
	}
	enc, err = proto.Marshal(vIdxList)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}
	if err := bucket.Put(key, enc); err != nil {
		return errors.Wrap(err, "failed to include indexed attestation in the historical bucket")
	}
	return nil
}

func (db *Store) pruneAttHistory(currentEpoch uint64, historySize uint64) error {
	pruneFromEpoch := int64(currentEpoch) - int64(historySize)
	if pruneFromEpoch <= 0 {
		return nil
	}

	return db.update(func(tx *bolt.Tx) error {
		attBucket := tx.Bucket(historicIndexedAttestationsBucket)
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		max := bytesutil.Bytes8(uint64(pruneFromEpoch))
		for k, _ := c.First(); k != nil && bytes.Compare(k[:8], max) <= 0; k, _ = c.Next() {
			if err := attBucket.Delete(k); err != nil {
				return errors.Wrap(err, "failed to delete indexed attestation from historical bucket")
			}
		}

		idxBucket := tx.Bucket(compressedIdxAttsBucket)
		c = tx.Bucket(compressedIdxAttsBucket).Cursor()
		for k, _ := c.First(); k != nil && bytes.Compare(k[:8], max) <= 0; k, _ = c.Next() {
			if err := idxBucket.Delete(k); err != nil {
				return errors.Wrap(err, "failed to delete indexed attestation from historical bucket")
			}
		}
		return nil
	})
}

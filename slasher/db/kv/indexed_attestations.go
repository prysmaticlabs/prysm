package kv

import (
	"bytes"
	"context"
	"fmt"
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
	"go.opencensus.io/trace"
)

func unmarshalIdxAtt(ctx context.Context, enc []byte) (*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.unmarshalIdxAtt")
	defer span.End()
	protoIdxAtt := &ethpb.IndexedAttestation{}
	err := proto.Unmarshal(enc, protoIdxAtt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoded indexed attestation")
	}
	return protoIdxAtt, nil
}

func unmarshalCompressedIdxAttList(ctx context.Context, enc []byte) (*slashpb.CompressedIdxAttList, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.unmarshalCompressedIdxAttList")
	defer span.End()
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
func (db *Store) IdxAttsForTargetFromID(ctx context.Context, targetEpoch uint64, validatorID uint64) ([]*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.IdxAttsForTargetFromID")
	defer span.End()
	var idxAtts []*ethpb.IndexedAttestation

	err := db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(compressedIdxAttsBucket)
		enc := bucket.Get(bytesutil.Bytes8(targetEpoch))
		if enc == nil {
			return nil
		}
		idToIdxAttsList, err := unmarshalCompressedIdxAttList(ctx, enc)
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
				att, err := unmarshalIdxAtt(ctx, enc)
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
func (db *Store) IdxAttsForTarget(ctx context.Context, targetEpoch uint64) ([]*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.IdxAttsForTarget")
	defer span.End()
	var idxAtts []*ethpb.IndexedAttestation
	key := bytesutil.Bytes8(targetEpoch)
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		for k, enc := c.Seek(key); k != nil && bytes.Equal(k[:8], key); k, _ = c.Next() {
			idxAtt, err := unmarshalIdxAtt(ctx, enc)
			if err != nil {
				return err
			}
			idxAtts = append(idxAtts, idxAtt)
		}
		return nil
	})
	return idxAtts, err
}

// IndexedAttestations --
func (db *Store) IndexedAttestations(ctx context.Context, targetEpoch uint64) ([]*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.IndexedAttestations")
	defer span.End()
	var idxAtts []*ethpb.IndexedAttestation
	key := bytesutil.Bytes8(targetEpoch)
	err := db.view(func(tx *bolt.Tx) error {
		rootsBkt := tx.Bucket(indexedAttestationsRootsByTargetBucket)
		attRoots := rootsBkt.Get(key)
		splitRoots := make([][]byte, 0)
		for i := 0; i < len(attRoots); i += 32 {
			splitRoots = append(splitRoots, attRoots[i:i+32])
		}
		attsBkt := tx.Bucket(indexedAttestationsBucket)
		for i := 0; i < len(splitRoots); i++ {
			enc := attsBkt.Get(splitRoots[i])
			idxAtt, err := unmarshalIdxAtt(ctx, enc)
			if err != nil {
				return err
			}
			idxAtts = append(idxAtts, idxAtt)
		}
		return nil
	})
	return idxAtts, err
}

// SaveIncomingIndexedAttestations --
func (db *Store) SaveIncomingIndexedAttestations(ctx context.Context, atts []*ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SaveIndexedAttestations")
	defer span.End()
	encoded := make([][]byte, len(atts))
	encodedRoots := make([][]byte, len(atts))
	for i := 0; i < len(encoded); i++ {
		enc, err := proto.Marshal(atts[i])
		if err != nil {
			return errors.Wrap(err, "failed to marshal")
		}
		encoded[i] = enc
		root := hashutil.Hash(enc)
		encodedRoots[i] = root[:]
	}
	return db.update(func(tx *bolt.Tx) error {
		rootsBkt := tx.Bucket(indexedAttestationsRootsByTargetBucket)
		attsBkt := tx.Bucket(indexedAttestationsBucket)

		for i := 0; i < len(atts); i++ {
			targetEpochKey := bytesutil.Bytes8(atts[i].Data.Target.Epoch)
			attRoots := rootsBkt.Get(targetEpochKey)
			if err := rootsBkt.Put(targetEpochKey, append(attRoots, encodedRoots[i]...)); err != nil {
				return err
			}
			if err := attsBkt.Put(encodedRoots[i], encoded[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveIncomingIndexedAttestation --
func (db *Store) SaveIncomingIndexedAttestation(ctx context.Context, att *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SaveIncomingIndexedAttestation")
	defer span.End()
	enc, err := proto.Marshal(att)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}
	root := hashutil.Hash(enc)
	return db.update(func(tx *bolt.Tx) error {
		rootsBkt := tx.Bucket(indexedAttestationsRootsByTargetBucket)
		attsBkt := tx.Bucket(indexedAttestationsBucket)

		targetEpochKey := bytesutil.Bytes8(att.Data.Target.Epoch)
		attRoots := rootsBkt.Get(targetEpochKey)
		if err := rootsBkt.Put(targetEpochKey, append(attRoots, root[:]...)); err != nil {
			return err
		}
		if err := attsBkt.Put(root[:], enc); err != nil {
			return err
		}
		return nil
	})
}

// LatestIndexedAttestationsTargetEpoch returns latest target epoch in db
// returns 0 if there is no indexed attestations in db.
func (db *Store) LatestIndexedAttestationsTargetEpoch(ctx context.Context) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.LatestIndexedAttestationsTargetEpoch")
	defer span.End()
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
func (db *Store) LatestValidatorIdx(ctx context.Context) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.LatestValidatorIdx")
	defer span.End()
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

// DoubleVotes looks up db for slashable attesting data that were preformed by the same validator.
func (db *Store) DoubleVotes(ctx context.Context, validatorIdx uint64, dataRoot []byte, origAtt *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.DoubleVotes")
	defer span.End()
	idxAtts, err := db.IdxAttsForTargetFromID(ctx, origAtt.Data.Target.Epoch, validatorIdx)
	if err != nil {
		return nil, err
	}
	if idxAtts == nil || len(idxAtts) == 0 {
		return nil, fmt.Errorf("can't check nil indexed attestation for double vote")
	}

	var idxAttsToSlash []*ethpb.IndexedAttestation
	for _, att := range idxAtts {
		if att.Data == nil {
			continue
		}
		root, err := hashutil.HashProto(att.Data)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(root[:], dataRoot) {
			idxAttsToSlash = append(idxAttsToSlash, att)
		}
	}

	var as []*ethpb.AttesterSlashing
	for _, idxAtt := range idxAttsToSlash {
		as = append(as, &ethpb.AttesterSlashing{
			Attestation_1: origAtt,
			Attestation_2: idxAtt,
		})
	}
	return as, nil
}

// HasIndexedAttestation accepts an epoch and validator id and returns true if the indexed attestation exists.
func (db *Store) HasIndexedAttestation(ctx context.Context, targetEpoch uint64, validatorID uint64) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.HasIndexedAttestation")
	defer span.End()
	key := bytesutil.Bytes8(targetEpoch)
	var hasAttestation bool
	// #nosec G104
	err := db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(compressedIdxAttsBucket)
		enc := bucket.Get(key)
		if enc == nil {
			return nil
		}
		iList, err := unmarshalCompressedIdxAttList(ctx, enc)
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
func (db *Store) SaveIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SaveIndexedAttestation")
	defer span.End()
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
		if err := saveCompressedIdxAttToEpochList(ctx, idxAttestation, tx); err != nil {
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
		if err = db.PruneAttHistory(ctx, idxAttestation.Data.Source.Epoch, wsPeriod); err != nil {
			return err
		}
	}
	return err
}

func saveCompressedIdxAttToEpochList(ctx context.Context, idxAttestation *ethpb.IndexedAttestation, tx *bolt.Tx) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.saveCompressedIdxAttToEpochList")
	defer span.End()
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
	compressedIdxAttList, err := unmarshalCompressedIdxAttList(ctx, enc)
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
func (db *Store) DeleteIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.DeleteIndexedAttestation")
	defer span.End()
	key := encodeEpochSig(idxAttestation.Data.Target.Epoch, idxAttestation.Signature)
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		enc := bucket.Get(key)
		if enc == nil {
			return nil
		}
		if err := removeIdxAttIndicesByEpochFromDB(ctx, idxAttestation, tx); err != nil {
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

func removeIdxAttIndicesByEpochFromDB(ctx context.Context, idxAttestation *ethpb.IndexedAttestation, tx *bolt.Tx) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.removeIdxAttIndicesByEpochFromDB")
	defer span.End()
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
	vIdxList, err := unmarshalCompressedIdxAttList(ctx, enc)
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

// PruneAttHistory removes all attestations from the DB older than the pruning epoch age.
func (db *Store) PruneAttHistory(ctx context.Context, currentEpoch uint64, pruningEpochAge uint64) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.pruneAttHistory")
	defer span.End()
	pruneFromEpoch := int64(currentEpoch) - int64(pruningEpochAge)
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

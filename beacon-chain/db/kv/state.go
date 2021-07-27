package kv

import (
	"bytes"
	"context"

	"github.com/golang/snappy"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/genesis"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	v1alpha "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	statepb "github.com/prysmaticlabs/prysm/proto/prysm/v2/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// State returns the saved state using block's signing root,
// this particular block was used to generate the state.
func (s *Store) State(ctx context.Context, blockRoot [32]byte) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.State")
	defer span.End()
	var st *statepb.BeaconState
	enc, err := s.stateBytes(ctx, blockRoot)
	if err != nil {
		return nil, err
	}

	if len(enc) == 0 {
		return nil, nil
	}

	// get the validator entries of the state
	valEntries, valErr := s.validatorEntries(ctx, blockRoot)
	if valErr != nil {
		return nil, valErr
	}

	st, err = s.createState(ctx, enc, valEntries)
	if err != nil {
		return nil, err
	}
	return v1.InitializeFromProtoUnsafe(st)
}

// GenesisState returns the genesis state in beacon chain.
func (s *Store) GenesisState(ctx context.Context) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.GenesisState")
	defer span.End()

	cached, err := genesis.State(params.BeaconConfig().ConfigName)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, err
	}
	span.AddAttributes(trace.BoolAttribute("cache_hit", cached != nil))
	if cached != nil {
		return cached, nil
	}

	var st *statepb.BeaconState
	if err = s.db.View(func(tx *bolt.Tx) error {
		// Retrieve genesis block's signing root from blocks bucket,
		// to look up what the genesis state is.
		bucket := tx.Bucket(blocksBucket)
		genesisBlockRoot := bucket.Get(genesisBlockRootKey)

		bucket = tx.Bucket(stateBucket)
		enc := bucket.Get(genesisBlockRoot)
		if enc == nil {
			return nil
		}

		// get the validator entries of the genesis state
		valEntries, valErr := s.validatorEntries(ctx, bytesutil.ToBytes32(genesisBlockRoot))
		if valErr != nil {
			return valErr
		}

		var crtErr error
		st, crtErr = s.createState(ctx, enc, valEntries)
		return crtErr
	}); err != nil {
		return nil, err
	}
	if st == nil {
		return nil, nil
	}
	return v1.InitializeFromProtoUnsafe(st)
}

// SaveState stores a state to the db using block's signing root which was used to generate the state.
func (s *Store) SaveState(ctx context.Context, st state.ReadOnlyBeaconState, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveState")
	defer span.End()
	ok, err := s.isStateValidatorMigrationOver()
	if err != nil {
		return err
	}
	if ok {
		return s.SaveStatesEfficient(ctx, []state.ReadOnlyBeaconState{st}, [][32]byte{blockRoot})
	}
	return s.SaveStates(ctx, []state.ReadOnlyBeaconState{st}, [][32]byte{blockRoot})
}

// SaveStates stores multiple states to the db using the provided corresponding roots.
func (s *Store) SaveStates(ctx context.Context, states []state.ReadOnlyBeaconState, blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveStates")
	defer span.End()
	if states == nil {
		return errors.New("nil state")
	}
	multipleEncs := make([][]byte, len(states))
	for i, st := range states {
		pbState, err := v1.ProtobufBeaconState(st.InnerStateUnsafe())
		if err != nil {
			return err
		}
		multipleEncs[i], err = encode(ctx, pbState)
		if err != nil {
			return err
		}
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateBucket)
		for i, rt := range blockRoots {
			indicesByBucket := createStateIndicesFromStateSlot(ctx, states[i].Slot())
			if err := updateValueForIndices(ctx, indicesByBucket, rt[:], tx); err != nil {
				return errors.Wrap(err, "could not update DB indices")
			}
			if err := bucket.Put(rt[:], multipleEncs[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveStatesEfficient stores multiple states to the db (new schema) using the provided corresponding roots.
func (s *Store) SaveStatesEfficient(ctx context.Context, states []state.ReadOnlyBeaconState, blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveStatesEfficient")
	defer span.End()
	if states == nil {
		return errors.New("nil state")
	}
	multipleEncs := make([][]byte, len(states))
	validatorsEntries := make(map[string]*v1alpha.Validator)    // It's a map to make sure that you store only new validator entries.
	validatorKeys := make([][]byte, len(states))                // For every state, this stores a compressed list of validator keys.
	realValidators := make([][]*v1alpha.Validator, len(states)) // It's temporary structure to restore state in memory after persisting it.
	for i, st := range states {
		pbState, err := v1.ProtobufBeaconState(st.InnerStateUnsafe())
		if err != nil {
			return err
		}
		// store the real validators to restore the in memory state later.
		realValidators[i] = pbState.Validators

		// yank out the validators and store them in separate table to save space.
		var hashes []byte
		for _, val := range pbState.Validators {
			// create the unique hash for that validator entry.
			//hash := hashutil.Hash(valBytes)
			hash, hashErr := val.HashTreeRoot()
			if hashErr != nil {
				return hashErr
			}
			hashes = append(hashes, hash[:]...)

			// note down the hash and the encoded validator entry
			hashStr := string(hash[:])
			validatorsEntries[hashStr] = val
		}

		// zero out the validators List from the state bucket so that it is not stored as part of it.
		pbState.Validators = make([]*v1alpha.Validator, 0)

		validatorKeys[i] = snappy.Encode(nil, hashes)
		multipleEncs[i], err = encode(ctx, pbState)
		if err != nil {
			return err
		}
	}

	if err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateBucket)
		valIdxBkt := tx.Bucket(blockRootValidatorHashesBucket)
		for i, rt := range blockRoots {
			indicesByBucket := createStateIndicesFromStateSlot(ctx, states[i].Slot())
			if err := updateValueForIndices(ctx, indicesByBucket, rt[:], tx); err != nil {
				return errors.Wrap(err, "could not update DB indices")
			}
			if err := bucket.Put(rt[:], multipleEncs[i]); err != nil {
				return err
			}
			if err := valIdxBkt.Put(rt[:], validatorKeys[i]); err != nil {
				return err
			}
		}

		// store the validator entries separately to save space.
		valBkt := tx.Bucket(stateValidatorsBucket)
		for hashStr, validatorEntry := range validatorsEntries {
			key := []byte(hashStr)
			// if the entry is not in the cache and not in the DB,
			// then insert it in the DB and add to the cache.
			if _, ok := s.validatorEntryCache.Get(key); !ok {
				validatorEntryCacheMiss.Inc()
				if valEntry := valBkt.Get(key); valEntry == nil {
					valBytes, encodeErr := encode(ctx, validatorEntry)
					if encodeErr != nil {
						return encodeErr
					}
					if putErr := valBkt.Put(key, valBytes); putErr != nil {
						return putErr
					}
					s.validatorEntryCache.Set(key, validatorEntry, int64(len(valBytes)))
				}
			} else {
				validatorEntryCacheHit.Inc()
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// restore the state in memory with the original validators
	for i, vals := range realValidators {
		pbState, err := v1.ProtobufBeaconState(states[i].InnerStateUnsafe())
		if err != nil {
			return err
		}
		pbState.Validators = vals
	}
	return nil
}

// HasState checks if a state by root exists in the db.
func (s *Store) HasState(ctx context.Context, blockRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasState")
	defer span.End()
	hasState := false
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(stateBucket)
		stBytes := bkt.Get(blockRoot[:])
		if len(stBytes) > 0 {
			hasState = true
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return hasState
}

// DeleteState by block root.
func (s *Store) DeleteState(ctx context.Context, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteState")
	defer span.End()

	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		genesisBlockRoot := bkt.Get(genesisBlockRootKey)

		bkt = tx.Bucket(checkpointBucket)
		enc := bkt.Get(finalizedCheckpointKey)
		checkpoint := &v1alpha.Checkpoint{}
		if enc == nil {
			checkpoint = &v1alpha.Checkpoint{Root: genesisBlockRoot}
		} else if err := decode(ctx, enc, checkpoint); err != nil {
			return err
		}

		blockBkt := tx.Bucket(blocksBucket)
		headBlkRoot := blockBkt.Get(headBlockRootKey)
		bkt = tx.Bucket(stateBucket)
		// Safe guard against deleting genesis, finalized, head state.
		if bytes.Equal(blockRoot[:], checkpoint.Root) || bytes.Equal(blockRoot[:], genesisBlockRoot) || bytes.Equal(blockRoot[:], headBlkRoot) {
			return errors.New("cannot delete genesis, finalized, or head state")
		}

		slot, err := s.slotByBlockRoot(ctx, tx, blockRoot[:])
		if err != nil {
			return err
		}
		indicesByBucket := createStateIndicesFromStateSlot(ctx, slot)
		if err := deleteValueForIndices(ctx, indicesByBucket, blockRoot[:], tx); err != nil {
			return errors.Wrap(err, "could not delete root for DB indices")
		}

		ok, err := s.isStateValidatorMigrationOver()
		if err != nil {
			return err
		}
		if ok {
			// remove the validator entry keys for the corresponding state.
			idxBkt := tx.Bucket(blockRootValidatorHashesBucket)
			compressedValidatorHashes := idxBkt.Get(blockRoot[:])
			err = idxBkt.Delete(blockRoot[:])
			if err != nil {
				return err
			}

			// remove the respective validator entries from the cache.
			if len(compressedValidatorHashes) == 0 {
				return errors.Errorf("invalid compressed validator keys length")
			}
			validatorHashes, sErr := snappy.Decode(nil, compressedValidatorHashes)
			if sErr != nil {
				return errors.Wrap(sErr, "failed to uncompress validator keys")
			}
			if len(validatorHashes)%hashLength != 0 {
				return errors.Errorf("invalid validator keys length: %d", len(validatorHashes))
			}
			for i := 0; i < len(validatorHashes); i += hashLength {
				key := validatorHashes[i : i+hashLength]
				s.validatorEntryCache.Del(key)
			}
		}

		return bkt.Delete(blockRoot[:])
	})
}

// DeleteStates by block roots.
func (s *Store) DeleteStates(ctx context.Context, blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteStates")
	defer span.End()

	for _, r := range blockRoots {
		if err := s.DeleteState(ctx, r); err != nil {
			return err
		}
	}

	return nil
}

// creates state from marshaled proto state bytes. Also add the validator entries retrieved
// from the validator bucket and complete the state construction.
func (s *Store) createState(ctx context.Context, enc []byte, validatorEntries []*v1alpha.Validator) (*statepb.BeaconState, error) {
	protoState := &statepb.BeaconState{}
	if err := decode(ctx, enc, protoState); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	ok, err := s.isStateValidatorMigrationOver()
	if err != nil {
		return protoState, err
	}
	if ok {
		protoState.Validators = validatorEntries
	}
	return protoState, nil
}

// Retrieve the validator entries for a given block root. These entries are stored in a
// separate bucket to reduce state size.
func (s *Store) validatorEntries(ctx context.Context, blockRoot [32]byte) ([]*v1alpha.Validator, error) {
	ok, err := s.isStateValidatorMigrationOver()
	if err != nil {
		return nil, err
	}
	if !ok {
		return make([]*v1alpha.Validator, 0), nil
	}
	ctx, span := trace.StartSpan(ctx, "BeaconDB.validatorEntries")
	defer span.End()
	var validatorEntries []*v1alpha.Validator
	err = s.db.View(func(tx *bolt.Tx) error {
		// get the validator keys from the index bucket
		idxBkt := tx.Bucket(blockRootValidatorHashesBucket)
		valKey := idxBkt.Get(blockRoot[:])
		if len(valKey) == 0 {
			return errors.Errorf("invalid compressed validator keys length")
		}

		// decompress the keys and check if they are of proper length.
		validatorKeys, sErr := snappy.Decode(nil, valKey)
		if sErr != nil {
			return errors.Wrap(sErr, "failed to uncompress validator keys")
		}
		if len(validatorKeys)%hashLength != 0 {
			return errors.Errorf("invalid validator keys length: %d", len(validatorKeys))
		}

		// get the corresponding validator entries from the validator bucket.
		valBkt := tx.Bucket(stateValidatorsBucket)
		for i := 0; i < len(validatorKeys); i += hashLength {
			key := validatorKeys[i : i+hashLength]
			// get the entry bytes from the cache or from the DB.
			v, ok := s.validatorEntryCache.Get(key)
			if ok {
				valEntry, vType := v.(*v1alpha.Validator)
				if vType {
					s.validatorEntryCache.Set(key, valEntry, int64(valEntry.SizeSSZ()))
					validatorEntries = append(validatorEntries, valEntry)
					validatorEntryCacheHit.Inc()
				} else {
					// this should never happen, but anyway it's good to bail out if one happens.
					return errors.New("validator cache does not have proper object type")
				}
			} else {
				// not in cache, so get it from the DB, decode it and add to the entry list.
				valEntryBytes := valBkt.Get(key)
				if len(valEntryBytes) == 0 {
					return errors.New("could not find validator entry")
				}
				encValEntry := &v1alpha.Validator{}
				decodeErr := decode(ctx, valEntryBytes, encValEntry)
				if decodeErr != nil {
					return errors.Wrap(decodeErr, "failed to decode validator entry keys")
				}
				validatorEntries = append(validatorEntries, encValEntry)
				validatorEntryCacheMiss.Inc()
			}
		}
		return nil
	})
	return validatorEntries, err
}

/// retrieves and assembles the state information from multiple buckets.
func (s *Store) stateBytes(ctx context.Context, blockRoot [32]byte) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.stateBytes")
	defer span.End()
	var dst []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(stateBucket)
		stBytes := bkt.Get(blockRoot[:])
		if len(stBytes) == 0 {
			return nil
		}
		// Due to https://github.com/boltdb/bolt/issues/204, we need to
		// allocate a byte slice separately in the transaction or there
		// is the possibility of a panic when accessing that particular
		// area of memory.
		dst = make([]byte, len(stBytes))
		copy(dst, stBytes)
		return nil
	})
	return dst, err
}

// slotByBlockRoot retrieves the corresponding slot of the input block root.
func (s *Store) slotByBlockRoot(ctx context.Context, tx *bolt.Tx, blockRoot []byte) (types.Slot, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.slotByBlockRoot")
	defer span.End()

	bkt := tx.Bucket(stateSummaryBucket)
	enc := bkt.Get(blockRoot)

	if enc == nil {
		// Fall back to check the block.
		bkt := tx.Bucket(blocksBucket)
		enc := bkt.Get(blockRoot)

		if enc == nil {
			// Fallback and check the state.
			bkt = tx.Bucket(stateBucket)
			enc = bkt.Get(blockRoot)
			if enc == nil {
				return 0, errors.New("state enc can't be nil")
			}
			// no need to construct the validator entries as it is not used here.
			s, err := s.createState(ctx, enc, nil)
			if err != nil {
				return 0, err
			}
			if s == nil {
				return 0, errors.New("state can't be nil")
			}
			return s.Slot, nil
		}
		b := &v1alpha.SignedBeaconBlock{}
		err := decode(ctx, enc, b)
		if err != nil {
			return 0, err
		}
		if err := helpers.VerifyNilBeaconBlock(wrapper.WrappedPhase0SignedBeaconBlock(b)); err != nil {
			return 0, err
		}
		return b.Block.Slot, nil
	}
	stateSummary := &statepb.StateSummary{}
	if err := decode(ctx, enc, stateSummary); err != nil {
		return 0, err
	}
	return stateSummary.Slot, nil
}

// HighestSlotStatesBelow returns the states with the highest slot below the input slot
// from the db. Ideally there should just be one state per slot, but given validator
// can double propose, a single slot could have multiple block roots and
// results states. This returns a list of states.
func (s *Store) HighestSlotStatesBelow(ctx context.Context, slot types.Slot) ([]state.ReadOnlyBeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HighestSlotStatesBelow")
	defer span.End()

	var best []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(stateSlotIndicesBucket)
		c := bkt.Cursor()
		for s, root := c.First(); s != nil; s, root = c.Next() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			key := bytesutil.BytesToSlotBigEndian(s)
			if root == nil {
				continue
			}
			if key >= slot {
				break
			}
			best = root
		}
		return nil
	}); err != nil {
		return nil, err
	}

	var st state.ReadOnlyBeaconState
	var err error
	if best != nil {
		st, err = s.State(ctx, bytesutil.ToBytes32(best))
		if err != nil {
			return nil, err
		}
	}
	if st == nil || st.IsNil() {
		st, err = s.GenesisState(ctx)
		if err != nil {
			return nil, err
		}
	}

	return []state.ReadOnlyBeaconState{st}, nil
}

// createStateIndicesFromStateSlot takes in a state slot and returns
// a map of bolt DB index buckets corresponding to each particular key for indices for
// data, such as (shard indices bucket -> shard 5).
func createStateIndicesFromStateSlot(ctx context.Context, slot types.Slot) map[string][]byte {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.createStateIndicesFromState")
	defer span.End()
	indicesByBucket := make(map[string][]byte)
	// Every index has a unique bucket for fast, binary-search
	// range scans for filtering across keys.
	buckets := [][]byte{
		stateSlotIndicesBucket,
	}

	indices := [][]byte{
		bytesutil.SlotToBytesBigEndian(slot),
	}
	for i := 0; i < len(buckets); i++ {
		indicesByBucket[string(buckets[i])] = indices[i]
	}
	return indicesByBucket
}

// CleanUpDirtyStates removes states in DB that falls to under archived point interval rules.
// Only following states would be kept:
// 1.) state_slot % archived_interval == 0. (e.g. archived_interval=2048, states with slot 2048, 4096... etc)
// 2.) archived_interval - archived_interval/3 < state_slot % archived_interval
//   (e.g. archived_interval=2048, states with slots after 1365).
//   This is to tolerate skip slots. Not every state lays on the boundary.
// 3.) state with current finalized root
// 4.) unfinalized States
func (s *Store) CleanUpDirtyStates(ctx context.Context, slotsPerArchivedPoint types.Slot) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB. CleanUpDirtyStates")
	defer span.End()

	f, err := s.FinalizedCheckpoint(ctx)
	if err != nil {
		return err
	}
	finalizedSlot, err := helpers.StartSlot(f.Epoch)
	if err != nil {
		return err
	}
	deletedRoots := make([][32]byte, 0)

	err = s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(stateSlotIndicesBucket)
		return bkt.ForEach(func(k, v []byte) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			finalizedChkpt := bytesutil.ToBytes32(f.Root) == bytesutil.ToBytes32(v)
			slot := bytesutil.BytesToSlotBigEndian(k)
			mod := slot % slotsPerArchivedPoint
			nonFinalized := slot > finalizedSlot

			// The following conditions cover 1, 2, 3 and 4 above.
			if mod != 0 && mod <= slotsPerArchivedPoint-slotsPerArchivedPoint/3 && !finalizedChkpt && !nonFinalized {
				deletedRoots = append(deletedRoots, bytesutil.ToBytes32(v))
			}
			return nil
		})
	})
	if err != nil {
		return err
	}

	// Length of to be deleted roots is 0. Nothing to do.
	if len(deletedRoots) == 0 {
		return nil
	}

	log.WithField("count", len(deletedRoots)).Info("Cleaning up dirty states")
	if err := s.DeleteStates(ctx, deletedRoots); err != nil {
		return err
	}

	return err
}

func (s *Store) isStateValidatorMigrationOver() (bool, error) {
	// if flag is enabled, then always follow the new code path.
	if featureconfig.Get().EnableHistoricalSpaceRepresentation {
		return true, nil
	}

	// if the flag is not enabled, but the migration is over, then
	// follow the new code path as if the flag is enabled.
	returnFlag := false
	if err := s.db.View(func(tx *bolt.Tx) error {
		mb := tx.Bucket(migrationsBucket)
		b := mb.Get(migrationStateValidatorsKey)
		returnFlag = bytes.Equal(b, migrationCompleted)
		return nil
	}); err != nil {
		return returnFlag, err
	}
	return returnFlag, nil

}

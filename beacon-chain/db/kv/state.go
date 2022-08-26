package kv

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/genesis"
	statenative "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	v1 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v1"
	v2 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v2"
	v3 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v3"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz/detect"
	"github.com/prysmaticlabs/prysm/v3/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// State returns the saved state using block's signing root,
// this particular block was used to generate the state.
func (s *Store) State(ctx context.Context, blockRoot [32]byte) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.State")
	defer span.End()
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

	return s.unmarshalState(ctx, enc, valEntries)
}

// StateOrError is just like State(), except it only returns a non-error response
// if the requested state is found in the database.
func (s *Store) StateOrError(ctx context.Context, blockRoot [32]byte) (state.BeaconState, error) {
	st, err := s.State(ctx, blockRoot)
	if err != nil {
		return nil, err
	}
	if st == nil || st.IsNil() {
		return nil, errors.Wrap(ErrNotFoundState, fmt.Sprintf("no state with blockroot=%#x", blockRoot))
	}
	return st, nil
}

// GenesisState returns the genesis state in beacon chain.
func (s *Store) GenesisState(ctx context.Context) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.GenesisState")
	defer span.End()

	cached, err := genesis.State(params.BeaconConfig().ConfigName)
	if err != nil {
		tracing.AnnotateError(span, err)
		return nil, err
	}
	span.AddAttributes(trace.BoolAttribute("cache_hit", cached != nil))
	if cached != nil {
		return cached, nil
	}

	var st state.BeaconState
	err = s.db.View(func(tx *bolt.Tx) error {
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
		st, err = s.unmarshalState(ctx, enc, valEntries)
		return crtErr
	})
	if err != nil {
		return nil, err
	}
	if st == nil || st.IsNil() {
		return nil, nil
	}
	return st, nil
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
		stateBytes, err := marshalState(ctx, st)
		if err != nil {
			return err
		}
		multipleEncs[i] = stateBytes
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateBucket)
		for i, rt := range blockRoots {
			if err := bucket.Put(rt[:], multipleEncs[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

type withValidators interface {
	GetValidators() []*ethpb.Validator
}

// SaveStatesEfficient stores multiple states to the db (new schema) using the provided corresponding roots.
func (s *Store) SaveStatesEfficient(ctx context.Context, states []state.ReadOnlyBeaconState, blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveStatesEfficient")
	defer span.End()
	if states == nil {
		return errors.New("nil state")
	}
	validatorKeys, validatorsEntries, err := getValidators(states)
	if err != nil {
		return err
	}

	if err := s.db.Update(func(tx *bolt.Tx) error {
		return s.saveStatesEfficientInternal(ctx, tx, blockRoots, states, validatorKeys, validatorsEntries)
	}); err != nil {
		return err
	}

	return nil
}

func getValidators(states []state.ReadOnlyBeaconState) ([][]byte, map[string]*ethpb.Validator, error) {
	validatorsEntries := make(map[string]*ethpb.Validator) // It's a map to make sure that you store only new validator entries.
	validatorKeys := make([][]byte, len(states))           // For every state, this stores a compressed list of validator keys.
	for i, st := range states {
		pb, ok := st.InnerStateUnsafe().(withValidators)
		if !ok {
			return nil, nil, errors.New("could not cast state to interface with GetValidators()")
		}
		validators := pb.GetValidators()

		// yank out the validators and store them in separate table to save space.
		var hashes []byte
		for _, val := range validators {
			// create the unique hash for that validator entry.
			hash, hashErr := val.HashTreeRoot()
			if hashErr != nil {
				return nil, nil, hashErr
			}
			hashes = append(hashes, hash[:]...)

			// note down the hash and the encoded validator entry
			hashStr := string(hash[:])
			validatorsEntries[hashStr] = val
		}
		validatorKeys[i] = snappy.Encode(nil, hashes)
	}
	return validatorKeys, validatorsEntries, nil
}

func (s *Store) saveStatesEfficientInternal(ctx context.Context, tx *bolt.Tx, blockRoots [][32]byte, states []state.ReadOnlyBeaconState, validatorKeys [][]byte, validatorsEntries map[string]*ethpb.Validator) error {
	bucket := tx.Bucket(stateBucket)
	valIdxBkt := tx.Bucket(blockRootValidatorHashesBucket)
	for i, rt := range blockRoots {
		// There is a gap when the states that are passed are used outside this
		// thread. But while storing the state object, we should not store the
		// validator entries.To bring the gap closer, we empty the validators
		// just before Put() and repopulate that state with original validators.
		// look at issue https://github.com/prysmaticlabs/prysm/issues/9262.
		switch rawType := states[i].InnerStateUnsafe().(type) {
		case *ethpb.BeaconState:
			var pbState *ethpb.BeaconState
			var err error
			if features.Get().EnableNativeState {
				pbState, err = statenative.ProtobufBeaconStatePhase0(rawType)
			} else {
				pbState, err = v1.ProtobufBeaconState(rawType)
			}
			if err != nil {
				return err
			}
			if pbState == nil {
				return errors.New("nil state")
			}
			valEntries := pbState.Validators
			pbState.Validators = make([]*ethpb.Validator, 0)
			encodedState, err := encode(ctx, pbState)
			if err != nil {
				return err
			}
			if err := bucket.Put(rt[:], encodedState); err != nil {
				return err
			}
			pbState.Validators = valEntries
			if err := valIdxBkt.Put(rt[:], validatorKeys[i]); err != nil {
				return err
			}
		case *ethpb.BeaconStateAltair:
			var pbState *ethpb.BeaconStateAltair
			var err error
			if features.Get().EnableNativeState {
				pbState, err = statenative.ProtobufBeaconStateAltair(rawType)
			} else {
				pbState, err = v2.ProtobufBeaconState(rawType)
			}
			if err != nil {
				return err
			}
			if pbState == nil {
				return errors.New("nil state")
			}
			valEntries := pbState.Validators
			pbState.Validators = make([]*ethpb.Validator, 0)
			rawObj, err := pbState.MarshalSSZ()
			if err != nil {
				return err
			}
			encodedState := snappy.Encode(nil, append(altairKey, rawObj...))
			if err := bucket.Put(rt[:], encodedState); err != nil {
				return err
			}
			pbState.Validators = valEntries
			if err := valIdxBkt.Put(rt[:], validatorKeys[i]); err != nil {
				return err
			}
		case *ethpb.BeaconStateBellatrix:
			var pbState *ethpb.BeaconStateBellatrix
			var err error
			if features.Get().EnableNativeState {
				pbState, err = statenative.ProtobufBeaconStateBellatrix(rawType)
			} else {
				pbState, err = v3.ProtobufBeaconState(rawType)
			}
			if err != nil {
				return err
			}
			if pbState == nil {
				return errors.New("nil state")
			}
			valEntries := pbState.Validators
			pbState.Validators = make([]*ethpb.Validator, 0)
			rawObj, err := pbState.MarshalSSZ()
			if err != nil {
				return err
			}
			encodedState := snappy.Encode(nil, append(bellatrixKey, rawObj...))
			if err := bucket.Put(rt[:], encodedState); err != nil {
				return err
			}
			pbState.Validators = valEntries
			if err := valIdxBkt.Put(rt[:], validatorKeys[i]); err != nil {
				return err
			}
		default:
			return errors.New("invalid state type")
		}
	}
	// store the validator entries separately to save space.
	return s.storeValidatorEntriesSeparately(ctx, tx, validatorsEntries)
}

func (s *Store) storeValidatorEntriesSeparately(ctx context.Context, tx *bolt.Tx, validatorsEntries map[string]*ethpb.Validator) error {
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
		finalized := &ethpb.Checkpoint{}
		if enc == nil {
			finalized = &ethpb.Checkpoint{Root: genesisBlockRoot}
		} else if err := decode(ctx, enc, finalized); err != nil {
			return err
		}

		enc = bkt.Get(justifiedCheckpointKey)
		justified := &ethpb.Checkpoint{}
		if enc == nil {
			justified = &ethpb.Checkpoint{Root: genesisBlockRoot}
		} else if err := decode(ctx, enc, justified); err != nil {
			return err
		}

		bkt = tx.Bucket(stateBucket)
		// Safeguard against deleting genesis, finalized, head state.
		if bytes.Equal(blockRoot[:], finalized.Root) || bytes.Equal(blockRoot[:], genesisBlockRoot) || bytes.Equal(blockRoot[:], justified.Root) {
			return ErrDeleteJustifiedAndFinalized
		}

		// Nothing to delete if state doesn't exist.
		enc = bkt.Get(blockRoot[:])
		if enc == nil {
			return nil
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
				validatorEntryCacheDelete.Inc()
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

// unmarshal state from marshaled proto state bytes to versioned state struct type.
func (s *Store) unmarshalState(_ context.Context, enc []byte, validatorEntries []*ethpb.Validator) (state.BeaconState, error) {
	var err error
	enc, err = snappy.Decode(nil, enc)
	if err != nil {
		return nil, err
	}

	switch {
	case hasBellatrixKey(enc):
		// Marshal state bytes to altair beacon state.
		protoState := &ethpb.BeaconStateBellatrix{}
		if err := protoState.UnmarshalSSZ(enc[len(bellatrixKey):]); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal encoding for altair")
		}
		ok, err := s.isStateValidatorMigrationOver()
		if err != nil {
			return nil, err
		}
		if ok {
			protoState.Validators = validatorEntries
		}
		return v3.InitializeFromProtoUnsafe(protoState)
	case hasAltairKey(enc):
		// Marshal state bytes to altair beacon state.
		protoState := &ethpb.BeaconStateAltair{}
		if err := protoState.UnmarshalSSZ(enc[len(altairKey):]); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal encoding for altair")
		}
		ok, err := s.isStateValidatorMigrationOver()
		if err != nil {
			return nil, err
		}
		if ok {
			protoState.Validators = validatorEntries
		}
		return v2.InitializeFromProtoUnsafe(protoState)
	default:
		// Marshal state bytes to phase 0 beacon state.
		protoState := &ethpb.BeaconState{}
		if err := protoState.UnmarshalSSZ(enc); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal encoding")
		}
		ok, err := s.isStateValidatorMigrationOver()
		if err != nil {
			return nil, err
		}
		if ok {
			protoState.Validators = validatorEntries
		}
		return v1.InitializeFromProtoUnsafe(protoState)
	}
}

// marshal versioned state from struct type down to bytes.
func marshalState(ctx context.Context, st state.ReadOnlyBeaconState) ([]byte, error) {
	switch st.InnerStateUnsafe().(type) {
	case *ethpb.BeaconState:
		rState, ok := st.InnerStateUnsafe().(*ethpb.BeaconState)
		if !ok {
			return nil, errors.New("non valid inner state")
		}
		return encode(ctx, rState)
	case *ethpb.BeaconStateAltair:
		rState, ok := st.InnerStateUnsafe().(*ethpb.BeaconStateAltair)
		if !ok {
			return nil, errors.New("non valid inner state")
		}
		if rState == nil {
			return nil, errors.New("nil state")
		}
		rawObj, err := rState.MarshalSSZ()
		if err != nil {
			return nil, err
		}
		return snappy.Encode(nil, append(altairKey, rawObj...)), nil
	case *ethpb.BeaconStateBellatrix:
		rState, ok := st.InnerStateUnsafe().(*ethpb.BeaconStateBellatrix)
		if !ok {
			return nil, errors.New("non valid inner state")
		}
		if rState == nil {
			return nil, errors.New("nil state")
		}
		rawObj, err := rState.MarshalSSZ()
		if err != nil {
			return nil, err
		}
		return snappy.Encode(nil, append(bellatrixKey, rawObj...)), nil
	default:
		return nil, errors.New("invalid inner state")
	}
}

// Retrieve the validator entries for a given block root. These entries are stored in a
// separate bucket to reduce state size.
func (s *Store) validatorEntries(ctx context.Context, blockRoot [32]byte) ([]*ethpb.Validator, error) {
	ok, err := s.isStateValidatorMigrationOver()
	if err != nil {
		return nil, err
	}
	if !ok {
		return make([]*ethpb.Validator, 0), nil
	}
	ctx, span := trace.StartSpan(ctx, "BeaconDB.validatorEntries")
	defer span.End()
	var validatorEntries []*ethpb.Validator
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
				valEntry, vType := v.(*ethpb.Validator)
				if vType {
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
				encValEntry := &ethpb.Validator{}
				decodeErr := decode(ctx, valEntryBytes, encValEntry)
				if decodeErr != nil {
					return errors.Wrap(decodeErr, "failed to decode validator entry keys")
				}
				validatorEntries = append(validatorEntries, encValEntry)
				validatorEntryCacheMiss.Inc()

				// should add here in cache
				s.validatorEntryCache.Set(key, encValEntry, int64(encValEntry.SizeSSZ()))
			}
		}
		return nil
	})
	return validatorEntries, err
}

// retrieves and assembles the state information from multiple buckets.
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

// CleanUpDirtyStates attempts to maintain the promise to save approximately <head slot / save state interval> states.
// To do that, we save about 1 state every eg 2048 slots (default slotsPerArchivedPoint value), calling the slot
// where the save happened the "save point". Due to skipped slots, there may not be a block at a multiple of 2048,
// in which case the saved state point will be at the slot where the last block was previously included in the interval.
// We don't want to delete the most recently finalized state, which is saved to the same database,
// and in long periods of non-finality, stategen may also write a state every 128 slots to aid in recovery.
// So we preserve:
//   1. any state where the slot number is a multiple of 2048 (slot % 2048 == 0)
//   2. any state with a slot number within 682 slots (2048/3) of a such a save point,
//   3. most recently finalized state
//   4. non-finalized states used by stategen
func (s *Store) CleanUpDirtyStates(ctx context.Context, slotsPerArchivedPoint types.Slot) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB. CleanUpDirtyStates")
	defer span.End()

	f, err := s.FinalizedCheckpoint(ctx)
	if err != nil {
		return err
	}
	finalizedSlot, err := slots.EpochStart(f.Epoch)
	if err != nil {
		return err
	}
	finalizedRoot := bytesutil.ToBytes32(f.Root)

	// We usually archive a state every 2048 slots. If a slot at with slot number % 2048 == 0 is skipped,
	// we will store the last un-skipped state instead. We don't know exactly how far back that state could be
	// from the skipped one, but a fudge factor of roughly 1/3 of the interval was chosen based on looking
	// at chain history for guidance. 1/3 of the default interval (2048) comes out to about 682 slots (or ~21 epochs).
	intervalTopThird := slotsPerArchivedPoint - slotsPerArchivedPoint/3

	seen := 0
	toDelete := make([][32]byte, 0)
	err = s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(stateBucket)
		bbkt := tx.Bucket(blocksBucket)
		return bkt.ForEach(func(k, _ []byte) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			seen += 1
			// If we could cheaply and easily read the first 50 or so bytes of the state,
			// we could pull the slot from the ssz-encoded bytes. But the state is very large (> 50MB) and
			// we need to read the entire thing to snappy.Decode it, so this code is betting that it's cheaper
			// to grab the corresponding block and decode that instead.
			enc := bbkt.Get(k[:32])
			if enc == nil {
				// the database is in an unexpected state, we should error out to prevent anything destructive.
				log.WithField("root", hexutil.Encode(k)).Error("Could not find block corresponding to saved state")
				return errors.Wrapf(errSavedStateMissingBlock, "root=%#x", k)
			}
			enc, err = snappy.Decode(nil, enc)
			if err != nil {
				return errors.Wrapf(err, "unable to snappy.Decode block with root=%#x", k)
			}
			slot, err := detect.SlotFromBlock(enc)
			if err != nil {
				return errors.Wrapf(err, "unable to extract slot from block with root=%#x", k)
			}
			mod := slot % slotsPerArchivedPoint
			// state is on an archive point, or within the final 1/3 of the interval (case 1 & 2)
			if mod == 0 || mod > intervalTopThird {
				return nil
			}

			// don't delete the state integrating the latest finalized block (case 3)
			if bytesutil.ToBytes32(k) == finalizedRoot {
				return nil
			}

			// don't delete states that haven't finalized yet - they may be in-use by the hot state cache (case 4)
			if slot > finalizedSlot {
				return nil
			}

			// delete everything else!
			toDelete = append(toDelete, bytesutil.ToBytes32(k))
			return nil
		})
	})
	if err != nil {
		return err
	}

	if len(toDelete) == 0 {
		log.WithField("db_total", seen).Info("No dirty states to clean up")
		return nil
	}

	log.WithField("db_total", seen).WithField("dirty", len(toDelete)).Info("Cleaning up dirty states")
	if err := s.DeleteStates(ctx, toDelete); err != nil {
		return err
	}

	return err
}

func (s *Store) isStateValidatorMigrationOver() (bool, error) {
	// if flag is enabled, then always follow the new code path.
	if features.Get().EnableHistoricalSpaceRepresentation {
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

package kv

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	light_client "github.com/OffchainLabs/prysm/v7/consensus-types/light-client"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	"github.com/OffchainLabs/prysm/v7/monitoring/tracing/trace"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

// keyForLightClientEpochData returns the storage key prefix of fork v. Forks that share a
// proto type keep their own key so a record decodes to its exact fork.
func keyForLightClientEpochData(v int) ([]byte, error) {
	switch v {
	case version.Gloas:
		return gloasKey, nil
	case version.Fulu:
		return fuluKey, nil
	case version.Electra:
		return ElectraKey, nil
	case version.Deneb:
		return denebKey, nil
	case version.Capella:
		return capellaKey, nil
	case version.Bellatrix:
		return bellatrixKey, nil
	case version.Altair:
		return altairKey, nil
	default:
		return nil, fmt.Errorf("unsupported light client epoch data version %s", version.String(v))
	}
}

// newLightClientEpochData returns an empty record of the proto type used by fork v.
func newLightClientEpochData(v int) (light_client.EpochData, error) {
	switch v {
	case version.Gloas:
		return &ethpb.LightClientEpochDataGloas{}, nil
	case version.Fulu, version.Electra:
		return &ethpb.LightClientEpochDataElectra{}, nil
	case version.Deneb:
		return &ethpb.LightClientEpochDataDeneb{}, nil
	case version.Capella:
		return &ethpb.LightClientEpochDataCapella{}, nil
	case version.Bellatrix, version.Altair:
		return &ethpb.LightClientEpochDataAltair{}, nil
	default:
		return nil, fmt.Errorf("unsupported light client epoch data version %s", version.String(v))
	}
}

// versionOfLightClientEpochData returns the fork whose key prefixes enc.
func versionOfLightClientEpochData(enc []byte) (int, error) {
	switch {
	case hasGloasKey(enc):
		return version.Gloas, nil
	case hasFuluKey(enc):
		return version.Fulu, nil
	case HasElectraKey(enc):
		return version.Electra, nil
	case hasDenebKey(enc):
		return version.Deneb, nil
	case hasCapellaKey(enc):
		return version.Capella, nil
	case hasBellatrixKey(enc):
		return version.Bellatrix, nil
	case hasAltairKey(enc):
		return version.Altair, nil
	default:
		return 0, errors.New("decoding of saved light client epoch data is unsupported")
	}
}

// SaveLightClientEpochData stores records in one transaction. Identical content is a no-op;
// different content fails the whole batch with ErrLightClientEpochDataConflict.
func (s *Store) SaveLightClientEpochData(ctx context.Context, records []light_client.EpochData) error {
	_, span := trace.StartSpan(ctx, "BeaconDB.SaveLightClientEpochData")
	defer span.End()

	// Validate and serialize outside the transaction.
	keys := make([][]byte, len(records))
	raws := make([][]byte, len(records))
	for i, r := range records {
		raw, err := encodeLightClientEpochData(r)
		if err != nil {
			return errors.Wrapf(err, "record %d", i)
		}
		keys[i] = bytesutil.Uint64ToBytesBigEndian(uint64(r.GetEpoch()))
		raws[i] = raw
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(lightClientEpochDataBucket)
		for i := range records {
			if err := ctx.Err(); err != nil {
				return err
			}
			existing := bkt.Get(keys[i])
			if existing == nil {
				if err := bkt.Put(keys[i], snappy.Encode(nil, raws[i])); err != nil {
					return err
				}
				continue
			}
			// snappy output is not canonical, so compare the uncompressed bytes.
			stored, err := snappy.Decode(nil, existing)
			if err != nil {
				return errors.Wrapf(err, "could not snappy decode stored light client epoch data for epoch %d", records[i].GetEpoch())
			}
			if !bytes.Equal(stored, raws[i]) {
				return errors.Wrapf(ErrLightClientEpochDataConflict, "epoch %d", records[i].GetEpoch())
			}
		}
		return nil
	})
}

// LightClientEpochData returns the record for epoch, or nil, nil when none is stored.
func (s *Store) LightClientEpochData(ctx context.Context, epoch primitives.Epoch) (light_client.EpochData, error) {
	_, span := trace.StartSpan(ctx, "BeaconDB.LightClientEpochData")
	defer span.End()

	var d light_client.EpochData
	err := s.db.View(func(tx *bolt.Tx) error {
		enc := tx.Bucket(lightClientEpochDataBucket).Get(bytesutil.Uint64ToBytesBigEndian(uint64(epoch)))
		if enc == nil {
			return nil
		}
		var err error
		d, err = decodeLightClientEpochData(enc)
		return err
	})
	if err != nil {
		return nil, err
	}
	if d != nil && d.GetEpoch() != epoch {
		return nil, fmt.Errorf("light client epoch data stored under epoch %d has epoch %d", epoch, d.GetEpoch())
	}
	return d, nil
}

// encodeLightClientEpochData returns the fork key followed by the SSZ of d, uncompressed.
func encodeLightClientEpochData(d light_client.EpochData) ([]byte, error) {
	if d == nil || !d.ProtoReflect().IsValid() {
		return nil, errors.New("nil light client epoch data")
	}
	if err := validateLightClientEpochDataContent(d); err != nil {
		return nil, err
	}
	v, err := light_client.EpochDataVersion(d)
	if err != nil {
		return nil, err
	}
	key, err := keyForLightClientEpochData(v)
	if err != nil {
		return nil, err
	}
	// A record decodes as the type of its fork key, so the type must match the content's fork.
	want, err := newLightClientEpochData(v)
	if err != nil {
		return nil, err
	}
	if reflect.TypeOf(d) != reflect.TypeOf(want) {
		return nil, fmt.Errorf("light client epoch data for %s must be %T, got %T", version.String(v), want, d)
	}
	// MarshalSSZ fills nil nested messages in place; marshal a copy to leave the input untouched.
	enc, err := proto.Clone(d).(light_client.EpochData).MarshalSSZ()
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal light client epoch data")
	}
	raw := make([]byte, 0, len(key)+len(enc))
	raw = append(raw, key...)
	return append(raw, enc...), nil
}

func decodeLightClientEpochData(enc []byte) (light_client.EpochData, error) {
	raw, err := snappy.Decode(nil, enc)
	if err != nil {
		return nil, errors.Wrap(err, "could not snappy decode light client epoch data")
	}
	v, err := versionOfLightClientEpochData(raw)
	if err != nil {
		return nil, err
	}
	key, err := keyForLightClientEpochData(v)
	if err != nil {
		return nil, err
	}
	d, err := newLightClientEpochData(v)
	if err != nil {
		return nil, err
	}
	if err := d.UnmarshalSSZ(raw[len(key):]); err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal %s light client epoch data", version.String(v))
	}
	return d, nil
}

func validateLightClientEpochDataContent(d light_client.EpochData) error {
	switch t := d.(type) {
	case *ethpb.LightClientEpochDataAltair:
		return validateLightClientEpochDataParts(t.GetBlockData(), t.GetBootstrapData().GetCurrentSyncCommittee())
	case *ethpb.LightClientEpochDataCapella:
		return validateLightClientEpochDataParts(t.GetBlockData(), t.GetBootstrapData().GetCurrentSyncCommittee())
	case *ethpb.LightClientEpochDataDeneb:
		return validateLightClientEpochDataParts(t.GetBlockData(), t.GetBootstrapData().GetCurrentSyncCommittee())
	case *ethpb.LightClientEpochDataElectra:
		return validateLightClientEpochDataParts(t.GetBlockData(), t.GetBootstrapData().GetCurrentSyncCommittee())
	case *ethpb.LightClientEpochDataGloas:
		return validateLightClientEpochDataParts(t.GetBlockData(), t.GetBootstrapData().GetCurrentSyncCommittee())
	default:
		return fmt.Errorf("unsupported light client epoch data type %T", d)
	}
}

// validateLightClientEpochDataParts rejects nil block_data entries, which MarshalSSZ cannot
// handle, and sync committees inside the record, which belong in the period bucket.
func validateLightClientEpochDataParts[B interface{ GetStateRoot() []byte }](blockData []B, committee []*ethpb.SyncCommittee) error {
	for i, b := range blockData {
		if len(b.GetStateRoot()) != fieldparams.RootLength {
			return fmt.Errorf("block_data[%d] is nil or has no state root", i)
		}
	}
	if len(committee) != 0 {
		return errors.New("current_sync_committee must be empty; sync committees are stored per period")
	}
	return nil
}

// SaveLightClientPeriodSyncCommittee stores a period's committee. The same committee again is
// a no-op; a different one fails with ErrLightClientPeriodSyncCommitteeConflict.
func (s *Store) SaveLightClientPeriodSyncCommittee(ctx context.Context, period uint64, committee *ethpb.SyncCommittee) error {
	_, span := trace.StartSpan(ctx, "BeaconDB.SaveLightClientPeriodSyncCommittee")
	defer span.End()

	if committee == nil {
		return errors.New("nil sync committee")
	}
	enc, err := committee.MarshalSSZ()
	if err != nil {
		return errors.Wrap(err, "could not marshal sync committee")
	}
	key := bytesutil.Uint64ToBytesBigEndian(period)
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(lightClientPeriodSyncCommitteeBucket)
		existing := bkt.Get(key)
		if existing == nil {
			return bkt.Put(key, enc)
		}
		if !bytes.Equal(existing, enc) {
			return errors.Wrapf(ErrLightClientPeriodSyncCommitteeConflict, "period %d", period)
		}
		return nil
	})
}

// LightClientPeriodSyncCommittee returns the committee of period, or nil, nil when none is stored.
func (s *Store) LightClientPeriodSyncCommittee(ctx context.Context, period uint64) (*ethpb.SyncCommittee, error) {
	_, span := trace.StartSpan(ctx, "BeaconDB.LightClientPeriodSyncCommittee")
	defer span.End()

	var committee *ethpb.SyncCommittee
	err := s.db.View(func(tx *bolt.Tx) error {
		enc := tx.Bucket(lightClientPeriodSyncCommitteeBucket).Get(bytesutil.Uint64ToBytesBigEndian(period))
		if enc == nil {
			return nil
		}
		committee = &ethpb.SyncCommittee{}
		if err := committee.UnmarshalSSZ(enc); err != nil {
			return errors.Wrap(err, "could not unmarshal sync committee")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return committee, nil
}

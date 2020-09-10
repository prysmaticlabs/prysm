package kv

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// the size of each data entry in bytes source epoch(8 bytes) signing root (32 bytes)
const entrySize = 40
const uint64Size = 8

type AttestationHistoryNew struct {
	TargetToSource     map[uint64]*HistoryData
	LatestEpochWritten uint64
}
type HistoryData struct {
	Source      uint64
	SigningRoot []byte
}

func unmarshalAttestationHistory(ctx context.Context, enc []byte) (*slashpb.AttestationHistory, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.unmarshalAttestationHistory")
	defer span.End()

	history := &slashpb.AttestationHistory{}
	if err := proto.Unmarshal(enc, history); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return history, nil
}

func sizeChecks(enc []byte) error {
	if enc == nil || len(enc) < uint64Size {
		return fmt.Errorf("encapsulated data size: %d is smaller then minimal size: %d", len(enc), uint64Size)
	}
	if (len(enc)-uint64Size)%entrySize != 0 {
		return fmt.Errorf("encapsulated data size: %d is not a multiple of entry size: %d", len(enc), entrySize)
	}
	return nil
}

func getLatestEpochWritten(ctx context.Context, enc []byte) (uint64, error) {
	if err := sizeChecks(enc); err != nil {
		return 0, err
	}

}

func setLatestEpochWritten(ctx context.Context, enc []byte, latestEpochWritten uint64) ([]byte, error) {
	if err := sizeChecks(enc); err != nil {
		return nil, err
	}
}
func getTargetData(ctx context.Context, enc []byte, target uint64) (*HistoryData, error) {
	if err := sizeChecks(enc); err != nil {
		return nil, err
	}
}
func setTargetData(ctx context.Context, enc []byte, target uint64, data HistoryData) ([]byte, error) {
	if err := sizeChecks(enc); err != nil {
		return nil, err
	}

}

func unmarshalAttestationHistoryNew(ctx context.Context, enc []byte) (*AttestationHistoryNew, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.unmarshalAttestationHistoryNew")
	defer span.End()
	history := &AttestationHistoryNew{}
	if (len(enc)-uint64Size)%entrySize != 0 {
		return nil, fmt.Errorf("encapsulated data size: %d is not a multiple of entry size: %d", len(enc), entrySize)
	}
	history.LatestEpochWritten = bytesutil.FromBytes8(enc[:uint64Size])

	mapSize := uint64((len(enc) - uint64Size) / entrySize)
	history.TargetToSource = make(map[uint64]*HistoryData, mapSize)
	for i := uint64(0); i < mapSize; i++ {
		hd := &HistoryData{
			Source:      bytesutil.FromBytes8(enc[i*entrySize+uint64Size : i*entrySize+2*uint64Size]),
			SigningRoot: enc[i*entrySize+2*uint64Size : i*entrySize+2*uint64Size+32],
		}
		history.TargetToSource[i] = hd
	}

	return history, nil
}

func marshalAttestationHistoryNew(ctx context.Context, attHis *AttestationHistoryNew) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.unmarshalAttestationHistoryNew")
	defer span.End()
	enc := make([]byte, len(attHis.TargetToSource)+uint64Size)
	copy(enc[:uint64Size], bytesutil.Uint64ToBytesLittleEndian(attHis.LatestEpochWritten))

	for i := uint64(0); i < uint64(len(attHis.TargetToSource)); i++ {
		copy(enc[uint64Size+i*entrySize:uint64Size+i*entrySize+uint64Size], bytesutil.Uint64ToBytesLittleEndian(attHis.TargetToSource[i].Source))
		copy(enc[uint64Size+i*entrySize+uint64Size:uint64Size+i*entrySize+uint64Size+32], attHis.TargetToSource[i].SigningRoot)
	}

	return enc, nil
}

// AttestationHistoryForPubKeys accepts an array of validator public keys and returns a mapping of corresponding attestation history.
func (store *Store) AttestationHistoryForPubKeys(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]*slashpb.AttestationHistory, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestationHistoryForPubKeys")
	defer span.End()

	if len(publicKeys) == 0 {
		return make(map[[48]byte]*slashpb.AttestationHistory), nil
	}

	var err error
	attestationHistoryForVals := make(map[[48]byte]*slashpb.AttestationHistory)
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicAttestationsBucket)
		for _, key := range publicKeys {
			enc := bucket.Get(key[:])
			var attestationHistory *slashpb.AttestationHistory
			if len(enc) == 0 {
				newMap := make(map[uint64]uint64)
				newMap[0] = params.BeaconConfig().FarFutureEpoch
				attestationHistory = &slashpb.AttestationHistory{
					TargetToSource: newMap,
				}
			} else {
				attestationHistory, err = unmarshalAttestationHistory(ctx, enc)
				if err != nil {
					return err
				}
			}
			attestationHistoryForVals[key] = attestationHistory
		}
		return nil
	})
	return attestationHistoryForVals, err
}

// SaveAttestationHistoryForPubKeys saves the attestation histories for the requested validator public keys.
func (store *Store) SaveAttestationHistoryForPubKeys(ctx context.Context, historyByPubKeys map[[48]byte]*slashpb.AttestationHistory) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveAttestationHistoryForPubKeys")
	defer span.End()

	encoded := make(map[[48]byte][]byte)
	for pubKey, history := range historyByPubKeys {
		enc, err := proto.Marshal(history)
		if err != nil {
			return errors.Wrap(err, "failed to encode attestation history")
		}
		encoded[pubKey] = enc
	}

	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicAttestationsBucket)
		for pubKey, encodedHistory := range encoded {
			if err := bucket.Put(pubKey[:], encodedHistory); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

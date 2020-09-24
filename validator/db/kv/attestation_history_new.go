package kv

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// the size of each data entry in bytes source epoch(8 bytes) signing root (32 bytes)
const uint64Size = 8
const latestEpochWrittenSize = uint64Size
const targetSize = uint64Size
const sourceSize = uint64Size
const signingRootSize = 32
const historySize = targetSize + sourceSize + signingRootSize
const minimalSize = latestEpochWrittenSize

// AttestationHistoryNew stores the historical attestation data needed
// for protection of validators.
type AttestationHistoryNew struct {
	TargetToSource     map[uint64]*HistoryData
	LatestEpochWritten uint64
}

// HistoryData stores the needed data to confirm if an attestation is slashable
// or repeated.
type HistoryData struct {
	Source      uint64
	SigningRoot []byte
}

func newAttestationHistoryArray(target uint64) []byte {
	enc := make([]byte, latestEpochWrittenSize+(target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize+historySize)
	return enc
}

func sizeChecks(enc []byte) error {
	if enc == nil || len(enc) < minimalSize {
		return fmt.Errorf("encapsulated data size: %d is smaller then minimal size: %d", len(enc), minimalSize)
	}
	if (len(enc)-minimalSize)%historySize != 0 {
		return fmt.Errorf("encapsulated data size: %d is not a multiple of entry size: %d", len(enc), historySize)
	}

	return nil
}

func getLatestEpochWritten(ctx context.Context, enc []byte) (uint64, error) {
	if err := sizeChecks(enc); err != nil {
		return 0, err
	}
	return bytesutil.FromBytes8(enc[:latestEpochWrittenSize]), nil
}

func setLatestEpochWritten(ctx context.Context, enc []byte, latestEpochWritten uint64) ([]byte, error) {
	if err := sizeChecks(enc); err != nil {
		return nil, err
	}
	copy(enc[:latestEpochWrittenSize], bytesutil.Uint64ToBytesLittleEndian(latestEpochWritten))
	return enc, nil
}

func getTargetData(ctx context.Context, enc []byte, target uint64) (*HistoryData, error) {
	if err := sizeChecks(enc); err != nil {
		return nil, err
	}
	cursor := (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize + latestEpochWrittenSize
	if uint64(len(enc)) < cursor+historySize {
		return nil, fmt.Errorf("encapsulated data size: %d is smaller then the requested target location: %d", len(enc), cursor+historySize)
	}
	history := &HistoryData{}

	history.Source = bytesutil.FromBytes8(enc[cursor : cursor+sourceSize])
	sr := make([]byte, 32)
	copy(enc[cursor+sourceSize:cursor+historySize], sr)
	history.SigningRoot = sr
	return history, nil
}

func setTargetData(ctx context.Context, enc []byte, target uint64, data *HistoryData) ([]byte, error) {
	if err := sizeChecks(enc); err != nil {
		return nil, err
	}
	cursor := latestEpochWrittenSize + (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize
	if uint64(len(enc)) < cursor+historySize {
		ext := make([]byte, cursor+historySize-uint64(len(enc)))
		enc = append(enc, ext...)
	}
	copy(enc[cursor:cursor+sourceSize], bytesutil.Uint64ToBytesLittleEndian(data.Source))
	copy(enc[cursor+sourceSize:cursor+sourceSize+signingRootSize], data.SigningRoot)
	return enc, nil
}

// AttestationHistoryNewForPubKeys accepts an array of validator public keys and returns a mapping of corresponding attestation history.
func (store *Store) AttestationHistoryNewForPubKeys(ctx context.Context, publicKeys [][48]byte) (map[[48]byte][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestationHistoryForPubKeys")
	defer span.End()

	if len(publicKeys) == 0 {
		return make(map[[48]byte][]byte), nil
	}

	var err error
	attestationHistoryForVals := make(map[[48]byte][]byte)
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		for _, key := range publicKeys {
			enc := bucket.Get(key[:])
			var attestationHistory []byte
			if len(enc) == 0 {
				attestationHistory = newAttestationHistoryArray(0)
			} else {
				attestationHistory = enc
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

// SaveAttestationHistoryNewForPubKeys saves the attestation histories for the requested validator public keys.
func (store *Store) SaveAttestationHistoryNewForPubKeys(ctx context.Context, historyByPubKeys map[[48]byte][]byte) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveAttestationHistoryForPubKeys")
	defer span.End()

	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		for pubKey, encodedHistory := range historyByPubKeys {
			if err := bucket.Put(pubKey[:], encodedHistory); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

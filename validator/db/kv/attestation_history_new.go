package kv

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

const (
	// The size of each data entry in bytes for the source epoch (8 bytes) and signing root (32 bytes).
	uint64Size             = 8
	latestEpochWrittenSize = uint64Size
	targetSize             = uint64Size
	sourceSize             = uint64Size
	signingRootSize        = 32
	historySize            = targetSize + sourceSize + signingRootSize
	minimalSize            = latestEpochWrittenSize
)

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

// EncHistoryData encapsulated history data.
type EncHistoryData []byte

func (hd EncHistoryData) assertSize() error {
	if hd == nil || len(hd) < minimalSize {
		return fmt.Errorf("encapsulated data size: %d is smaller then minimal size: %d", len(hd), minimalSize)
	}
	if (len(hd)-minimalSize)%historySize != 0 {
		return fmt.Errorf("encapsulated data size: %d is not a multiple of entry size: %d", len(hd), historySize)
	}
	return nil
}

func newAttestationHistoryArray(target uint64) EncHistoryData {
	enc := make(EncHistoryData, latestEpochWrittenSize+(target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize+historySize)
	return enc
}

func (hd EncHistoryData) getLatestEpochWritten(ctx context.Context) (uint64, error) {
	if err := hd.assertSize(); err != nil {
		return 0, err
	}
	return bytesutil.FromBytes8(hd[:latestEpochWrittenSize]), nil
}

func (hd EncHistoryData) setLatestEpochWritten(ctx context.Context, latestEpochWritten uint64) (EncHistoryData, error) {
	if err := hd.assertSize(); err != nil {
		return nil, err
	}
	copy(hd[:latestEpochWrittenSize], bytesutil.Uint64ToBytesLittleEndian(latestEpochWritten))
	return hd, nil
}

func (hd EncHistoryData) getTargetData(ctx context.Context, target uint64) (*HistoryData, error) {
	if err := hd.assertSize(); err != nil {
		return nil, err
	}
	// cursor for the location to read target epoch from.
	// modulus of target epoch  X weak subjectivity period in order to have maximum size to the encapsulated data array.
	cursor := (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize + latestEpochWrittenSize
	if uint64(len(hd)) < cursor+historySize {
		return nil, fmt.Errorf("encapsulated data size: %d is smaller then the requested target location: %d", len(hd), cursor+historySize)
	}
	history := &HistoryData{}

	history.Source = bytesutil.FromBytes8(hd[cursor : cursor+sourceSize])
	sr := make([]byte, 32)
	copy(hd[cursor+sourceSize:cursor+historySize], sr)
	history.SigningRoot = sr
	return history, nil
}

func (hd EncHistoryData) setTargetData(ctx context.Context, target uint64, historyData *HistoryData) (EncHistoryData, error) {
	if err := hd.assertSize(); err != nil {
		return nil, err
	}
	// Cursor for the location to write target epoch to.
	// Modulus of target epoch  X weak subjectivity period in order to have maximum size to the encapsulated data array.
	cursor := latestEpochWrittenSize + (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize
	if uint64(len(hd)) < cursor+historySize {
		ext := make([]byte, cursor+historySize-uint64(len(hd)))
		hd = append(hd, ext...)
	}
	copy(hd[cursor:cursor+sourceSize], bytesutil.Uint64ToBytesLittleEndian(historyData.Source))
	copy(hd[cursor+sourceSize:cursor+sourceSize+signingRootSize], historyData.SigningRoot)
	return hd, nil
}

// AttestationHistoryNewForPubKeys accepts an array of validator public keys and returns a mapping of corresponding attestation history.
func (store *Store) AttestationHistoryNewForPubKeys(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]EncHistoryData, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestationHistoryForPubKeys")
	defer span.End()

	if len(publicKeys) == 0 {
		return make(map[[48]byte]EncHistoryData), nil
	}

	var err error
	attestationHistoryForVals := make(map[[48]byte]EncHistoryData)
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
func (store *Store) SaveAttestationHistoryNewForPubKeys(ctx context.Context, historyByPubKeys map[[48]byte]EncHistoryData) error {
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

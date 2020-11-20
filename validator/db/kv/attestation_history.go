package kv

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
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

func (h *HistoryData) IsEmpty() bool {
	if h == (*HistoryData)(nil) {
		return true
	}
	if h.Source == params.BeaconConfig().FarFutureEpoch {
		return true
	}
	return false
}

func emptyHistoryData() *HistoryData {
	h := &HistoryData{Source: params.BeaconConfig().FarFutureEpoch, SigningRoot: bytesutil.PadTo([]byte{}, 32)}
	return h
}

// NewAttestationHistoryArray creates a new encapsulated attestation history byte array
// sized by the latest epoch written.
func NewAttestationHistoryArray(target uint64) EncHistoryData {
	relativeTarget := target % params.BeaconConfig().WeakSubjectivityPeriod
	historyDataSize := (relativeTarget + 1) * historySize
	arraySize := latestEpochWrittenSize + historyDataSize
	en := make(EncHistoryData, arraySize)
	enc := en
	ctx := context.Background()
	var err error
	for i := uint64(0); i <= target%params.BeaconConfig().WeakSubjectivityPeriod; i++ {
		enc, err = enc.SetTargetData(ctx, i, emptyHistoryData())
		if err != nil {
			log.WithError(err).Error("Failed to set empty target data")
		}
	}
	return enc
}

func (hd EncHistoryData) GetLatestEpochWritten(ctx context.Context) (uint64, error) {
	if err := hd.assertSize(); err != nil {
		return 0, err
	}
	return bytesutil.FromBytes8(hd[:latestEpochWrittenSize]), nil
}

func (hd EncHistoryData) SetLatestEpochWritten(ctx context.Context, latestEpochWritten uint64) (EncHistoryData, error) {
	if err := hd.assertSize(); err != nil {
		return nil, err
	}
	copy(hd[:latestEpochWrittenSize], bytesutil.Uint64ToBytesLittleEndian(latestEpochWritten))
	return hd, nil
}

func (hd EncHistoryData) GetTargetData(ctx context.Context, target uint64) (*HistoryData, error) {
	if err := hd.assertSize(); err != nil {
		return nil, err
	}
	// Cursor for the location to read target epoch from.
	// Modulus of target epoch  X weak subjectivity period in order to have maximum size to the encapsulated data array.
	cursor := (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize + latestEpochWrittenSize
	if uint64(len(hd)) < cursor+historySize {
		return nil, nil
	}
	history := &HistoryData{}
	history.Source = bytesutil.FromBytes8(hd[cursor : cursor+sourceSize])
	sr := make([]byte, 32)
	copy(sr, hd[cursor+sourceSize:cursor+historySize])
	history.SigningRoot = sr
	return history, nil
}

func (hd EncHistoryData) SetTargetData(ctx context.Context, target uint64, historyData *HistoryData) (EncHistoryData, error) {
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

// UpdateHistoryForAttestation returns a modified attestation history with specified target+epoch pairs marked
// as attested for. This is used to prevent the validator client from signing any slashable attestations.
func (hd EncHistoryData) UpdateHistoryForAttestation(
	ctx context.Context,
	sourceEpoch,
	targetEpoch uint64,
	signingRoot [32]byte,
) (EncHistoryData, error) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	latestEpochWritten, err := hd.GetLatestEpochWritten(ctx)
	if err != nil {
		return EncHistoryData{}, errors.Wrap(err, "could not get latest epoch written from history")
	}
	currentHD := hd
	if targetEpoch > latestEpochWritten {
		// If the target epoch to mark is ahead of latest written epoch, override the old targets and mark the requested epoch.
		// Limit the overwriting to one weak subjectivity period as further is not needed.
		maxToWrite := latestEpochWritten + wsPeriod
		for i := latestEpochWritten + 1; i < targetEpoch && i <= maxToWrite; i++ {
			newHD, err := hd.SetTargetData(ctx, i%wsPeriod, &HistoryData{Source: params.BeaconConfig().FarFutureEpoch})
			if err != nil {
				return EncHistoryData{}, errors.Wrap(err, "could not set target data")
			}
			currentHD = newHD
		}
		newHD, err := currentHD.SetLatestEpochWritten(ctx, targetEpoch)
		if err != nil {
			return EncHistoryData{}, errors.Wrap(err, "could not set latest epoch written")
		}
		currentHD = newHD
	}
	newHD, err := currentHD.SetTargetData(ctx, targetEpoch%wsPeriod, &HistoryData{Source: sourceEpoch, SigningRoot: signingRoot[:]})
	if err != nil {
		return EncHistoryData{}, errors.Wrap(err, "could not set target data")
	}
	return newHD, nil
}

// AttestationHistoryForPubKeys accepts an array of validator public keys and returns a mapping of corresponding attestation history.
func (store *Store) AttestationHistoryForPubKeys(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]EncHistoryData, error) {
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
			if ctx.Err() != nil {
				return ctx.Err()
			}
			enc := bucket.Get(key[:])
			var attestationHistory EncHistoryData
			if len(enc) == 0 {
				attestationHistory = NewAttestationHistoryArray(0)
			} else {
				attestationHistory = enc
			}
			attestationHistoryForVals[key] = attestationHistory
		}
		return nil
	})
	for pk, ah := range attestationHistoryForVals {
		ehd := make(EncHistoryData, len(ah))
		copy(ehd, ah)
		attestationHistoryForVals[pk] = ehd
	}
	return attestationHistoryForVals, err
}

// AttestationHistoryForPubKey fetches the attestation history for a public key.
func (store *Store) AttestationHistoryForPubKey(ctx context.Context, publicKey [48]byte) (EncHistoryData, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestationHistoryForPubKey")
	defer span.End()

	var err error
	attestingHistory := NewAttestationHistoryArray(0)
	err = store.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		enc := bucket.Get(publicKey[:])
		if len(enc) != 0 {
			// Copy to prevent internal array reference being overwritten by boltdb.
			copy(attestingHistory, enc)
		}
		return nil
	})
	return attestingHistory, err
}

// SaveAttestationHistoryForPubKeys saves the attestation histories for the requested validator public keys.
func (store *Store) SaveAttestationHistoryForPubKeys(ctx context.Context, historyByPubKeys map[[48]byte]EncHistoryData) error {
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

// SaveAttestationHistoryForPubKey saves the attestation history for the requested validator public key.
func (store *Store) SaveAttestationHistoryForPubKey(ctx context.Context, pubKey [48]byte, history EncHistoryData) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveAttestationHistoryForPubKey")
	defer span.End()
	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		return bucket.Put(pubKey[:], history)
	})
	return err
}

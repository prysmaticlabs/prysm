package kv

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

var ErrWrongSize = errors.New("wrong data length for min max span byte array")
var highestObservedValidatorIdx uint64

// GetValidatorSpan unmarshal a span from an encoded, flattened array.
func (db *Store) GetValidatorSpan(ctx context.Context, spans []byte, validatorIdx uint64) (types.Span, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.getValidatorSpan")
	defer span.End()

	r := types.Span{}
	if len(spans)%spannerEncodedLength != 0 {
		return r, ErrWrongSize
	}
	origLength := uint64(len(spans)) / spannerEncodedLength
	requestedLength := validatorIdx + 1
	if origLength < requestedLength {
		return r, nil
	}
	cursor := validatorIdx * spannerEncodedLength
	r.MinSpan = bytesutil.FromBytes2(spans[cursor : cursor+2])
	r.MaxSpan = bytesutil.FromBytes2(spans[cursor+2 : cursor+4])
	sigB := [2]byte{}
	copy(sigB[:], spans[cursor+4:cursor+6])
	r.SigBytes = sigB
	r.HasAttested = bytesutil.ToBool(spans[cursor+6])
	return r, nil
}

// SetValidatorSpan marshal a validator span into an encoded, flattened array.
func (db *Store) SetValidatorSpan(ctx context.Context, spans []byte, validatorIdx uint64, newSpan types.Span) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.setValidatorSpan")
	defer span.End()

	if len(spans)%spannerEncodedLength != 0 {
		return nil, errors.New("wrong data length for min max span byte array")
	}
	if highestObservedValidatorIdx < validatorIdx {
		highestObservedValidatorIdx = validatorIdx
	}
	if len(spans) == 0 {
		requestedLength := highestObservedValidatorIdx*spannerEncodedLength + spannerEncodedLength
		b := make([]byte, requestedLength, requestedLength)
		spans = b

	}
	cursor := validatorIdx * spannerEncodedLength
	endCursor := cursor + spannerEncodedLength
	spansLength := uint64(len(spans))
	if endCursor > spansLength {
		diff := endCursor - spansLength
		b := make([]byte, diff, diff)
		spans = append(spans, b...)
	}
	enc := marshalSpan(newSpan)
	copy(spans[cursor:], enc)

	return spans, nil
}

// EpochSpans accepts epoch and returns the corresponding spans byte array
// for slashing detection.
// Returns span byte array, and error in case of db error.
// returns empty byte array if no entry for this epoch exists in db.
func (db *Store) EpochSpans(ctx context.Context, epoch uint64) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.EpochSpans")
	defer span.End()

	var err error
	var spans []byte
	err = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucketNew)
		if b == nil {
			return nil
		}
		spans = b.Get(bytesutil.Bytes8(epoch))
		return nil
	})
	if spans == nil {
		spans = []byte{}
	}
	return spans, err
}

// SaveEpochSpansMap accepts a epoch and span byte array and writes it to disk.
func (db *Store) SaveEpochSpans(ctx context.Context, epoch uint64, spans []byte) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.SaveEpochSpans")
	defer span.End()

	if len(spans)%spannerEncodedLength != 0 {
		return ErrWrongSize
	}
	return db.update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(validatorsMinMaxSpanBucketNew)
		if err != nil {
			return err
		}
		return b.Put(bytesutil.Bytes8(epoch), spans)
	})
}

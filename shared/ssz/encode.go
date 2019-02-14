package ssz

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
)

const lengthBytes = 4

// Encodable defines the interface for support ssz encoding.
type Encodable interface {
	EncodeSSZ(io.Writer) error
	EncodeSSZSize() (uint32, error)
}

// Encode encodes val and output the result into w.
func Encode(w io.Writer, val interface{}) error {
	eb := &encbuf{}
	if err := eb.encode(val); err != nil {
		return err
	}
	return eb.toWriter(w)
}

// EncodeSize returns the target encoding size without doing the actual encoding.
// This is an optional pass. You don't need to call this before the encoding unless you
// want to know the output size first.
func EncodeSize(val interface{}) (uint32, error) {
	return encodeSize(val)
}

type encbuf struct {
	str []byte
}

func (w *encbuf) encode(val interface{}) error {
	if val == nil {
		return newEncodeError("untyped nil is not supported", nil)
	}
	rval := reflect.ValueOf(val)
	sszUtils, err := cachedSSZUtils(rval.Type())
	if err != nil {
		return newEncodeError(fmt.Sprint(err), rval.Type())
	}
	if err = sszUtils.encoder(rval, w); err != nil {
		return newEncodeError(fmt.Sprint(err), rval.Type())
	}
	return nil
}

func encodeSize(val interface{}) (uint32, error) {
	if val == nil {
		return 0, newEncodeError("untyped nil is not supported", nil)
	}
	rval := reflect.ValueOf(val)
	sszUtils, err := cachedSSZUtils(rval.Type())
	if err != nil {
		return 0, newEncodeError(fmt.Sprint(err), rval.Type())
	}
	var size uint32
	if size, err = sszUtils.encodeSizer(rval); err != nil {
		return 0, newEncodeError(fmt.Sprint(err), rval.Type())
	}
	return size, nil

}

func (w *encbuf) toWriter(out io.Writer) error {
	_, err := out.Write(w.str)
	return err
}

func makeEncoder(typ reflect.Type) (encoder, encodeSizer, error) {
	kind := typ.Kind()
	switch {
	case kind == reflect.Bool:
		return encodeBool, func(reflect.Value) (uint32, error) { return 1, nil }, nil
	case kind == reflect.Uint8:
		return encodeUint8, func(reflect.Value) (uint32, error) { return 1, nil }, nil
	case kind == reflect.Uint16:
		return encodeUint16, func(reflect.Value) (uint32, error) { return 2, nil }, nil
	case kind == reflect.Uint32:
		return encodeUint32, func(reflect.Value) (uint32, error) { return 4, nil }, nil
	case kind == reflect.Int32:
		return encodeInt32, func(reflect.Value) (uint32, error) { return 4, nil }, nil
	case kind == reflect.Uint64:
		return encodeUint64, func(reflect.Value) (uint32, error) { return 8, nil }, nil
	case kind == reflect.Slice && typ.Elem().Kind() == reflect.Uint8:
		return makeBytesEncoder()
	case kind == reflect.Slice:
		return makeSliceEncoder(typ)
	case kind == reflect.Array && typ.Elem().Kind() == reflect.Uint8:
		return makeByteArrayEncoder()
	case kind == reflect.Array:
		return makeSliceEncoder(typ)
	case kind == reflect.Struct:
		return makeStructEncoder(typ)
	case kind == reflect.Ptr:
		return makePtrEncoder(typ)
	default:
		return nil, nil, fmt.Errorf("type %v is not serializable", typ)
	}
}

func encodeBool(val reflect.Value, w *encbuf) error {
	if val.Bool() {
		w.str = append(w.str, uint8(1))
	} else {
		w.str = append(w.str, uint8(0))
	}
	return nil
}

func encodeUint8(val reflect.Value, w *encbuf) error {
	v := val.Uint()
	w.str = append(w.str, uint8(v))
	return nil
}

func encodeUint16(val reflect.Value, w *encbuf) error {
	v := val.Uint()
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(v))
	w.str = append(w.str, b...)
	return nil
}

func encodeUint32(val reflect.Value, w *encbuf) error {
	v := val.Uint()
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(v))
	w.str = append(w.str, b...)
	return nil
}

func encodeInt32(val reflect.Value, w *encbuf) error {
	v := val.Int()
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(v))
	w.str = append(w.str, b...)
	return nil
}

func encodeUint64(val reflect.Value, w *encbuf) error {
	v := val.Uint()
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(v))
	w.str = append(w.str, b...)
	return nil
}

func makeBytesEncoder() (encoder, encodeSizer, error) {
	encoder := func(val reflect.Value, w *encbuf) error {
		b := val.Bytes()
		sizeEnc := make([]byte, lengthBytes)
		if len(val.Bytes()) >= 2<<32 {
			return errors.New("bytes oversize")
		}
		binary.LittleEndian.PutUint32(sizeEnc, uint32(len(b)))
		w.str = append(w.str, sizeEnc...)
		w.str = append(w.str, val.Bytes()...)
		return nil
	}
	encodeSizer := func(val reflect.Value) (uint32, error) {
		if len(val.Bytes()) >= 2<<32 {
			return 0, errors.New("bytes oversize")
		}
		return lengthBytes + uint32(len(val.Bytes())), nil
	}
	return encoder, encodeSizer, nil
}

func makeByteArrayEncoder() (encoder, encodeSizer, error) {
	encoder := func(val reflect.Value, w *encbuf) error {
		if !val.CanAddr() {
			// Slice requires the value to be addressable.
			// Make it addressable by copying.
			copyVal := reflect.New(val.Type()).Elem()
			copyVal.Set(val)
			val = copyVal
		}
		sizeEnc := make([]byte, lengthBytes)
		if val.Len() >= 2<<32 {
			return errors.New("bytes oversize")
		}
		binary.LittleEndian.PutUint32(sizeEnc, uint32(val.Len()))
		w.str = append(w.str, sizeEnc...)
		w.str = append(w.str, val.Slice(0, val.Len()).Bytes()...)
		return nil
	}
	encodeSizer := func(val reflect.Value) (uint32, error) {
		if val.Len() >= 2<<32 {
			return 0, errors.New("bytes oversize")
		}
		return lengthBytes + uint32(val.Len()), nil
	}
	return encoder, encodeSizer, nil
}

func makeSliceEncoder(typ reflect.Type) (encoder, encodeSizer, error) {
	elemSSZUtils, err := cachedSSZUtilsNoAcquireLock(typ.Elem())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get ssz utils: %v", err)
	}
	encoder := func(val reflect.Value, w *encbuf) error {
		origBufSize := len(w.str)
		totalSizeEnc := make([]byte, lengthBytes)
		w.str = append(w.str, totalSizeEnc...)
		for i := 0; i < val.Len(); i++ {
			if err := elemSSZUtils.encoder(val.Index(i), w); err != nil {
				return fmt.Errorf("failed to encode element of slice/array: %v", err)
			}
		}
		totalSize := len(w.str) - lengthBytes - origBufSize
		if totalSize >= 2<<32 {
			return errors.New("slice oversize")
		}
		binary.LittleEndian.PutUint32(totalSizeEnc, uint32(totalSize))
		copy(w.str[origBufSize:origBufSize+lengthBytes], totalSizeEnc)
		return nil
	}
	encodeSizer := func(val reflect.Value) (uint32, error) {
		if val.Len() == 0 {
			return lengthBytes, nil
		}
		elemSize, err := elemSSZUtils.encodeSizer(val.Index(0))
		if err != nil {
			return 0, errors.New("failed to get encode size of element of slice/array")
		}
		return lengthBytes + elemSize*uint32(val.Len()), nil
	}
	return encoder, encodeSizer, nil
}

func makeStructEncoder(typ reflect.Type) (encoder, encodeSizer, error) {
	fields, err := structFields(typ)
	if err != nil {
		return nil, nil, err
	}
	encoder := func(val reflect.Value, w *encbuf) error {
		origBufSize := len(w.str)
		totalSizeEnc := make([]byte, lengthBytes)
		w.str = append(w.str, totalSizeEnc...)
		for _, f := range fields {
			if err := f.sszUtils.encoder(val.Field(f.index), w); err != nil {
				return fmt.Errorf("failed to encode field of struct: %v", err)
			}
		}
		totalSize := len(w.str) - lengthBytes - origBufSize
		if totalSize >= 2<<32 {
			return errors.New("struct oversize")
		}
		binary.LittleEndian.PutUint32(totalSizeEnc, uint32(totalSize))
		copy(w.str[origBufSize:origBufSize+lengthBytes], totalSizeEnc)
		return nil
	}
	encodeSizer := func(val reflect.Value) (uint32, error) {
		totalSize := uint32(0)
		for _, f := range fields {
			fieldSize, err := f.sszUtils.encodeSizer(val.Field(f.index))
			if err != nil {
				return 0, fmt.Errorf("failed to get encode size for field of struct: %v", err)
			}
			totalSize += fieldSize
		}
		return lengthBytes + totalSize, nil
	}
	return encoder, encodeSizer, nil
}

func makePtrEncoder(typ reflect.Type) (encoder, encodeSizer, error) {
	elemSSZUtils, err := cachedSSZUtilsNoAcquireLock(typ.Elem())
	if err != nil {
		return nil, nil, err
	}

	// After considered the use case in Prysm, we've decided that:
	// - We assume we will only encode/decode pointer of array, slice or struct.
	// - The encoding for nil pointer shall be 0x00000000.
	encoder := func(val reflect.Value, w *encbuf) error {
		if val.IsNil() {
			totalSizeEnc := make([]byte, lengthBytes)
			w.str = append(w.str, totalSizeEnc...)
			return nil
		}
		return elemSSZUtils.encoder(val.Elem(), w)
	}

	encodeSizer := func(val reflect.Value) (uint32, error) {
		if val.IsNil() {
			return lengthBytes, nil
		}
		return elemSSZUtils.encodeSizer(val.Elem())
	}

	return encoder, encodeSizer, nil
}

// encodeError is what gets reported to the encoder user in error case.
type encodeError struct {
	msg string
	typ reflect.Type
}

func newEncodeError(msg string, typ reflect.Type) *encodeError {
	return &encodeError{msg, typ}
}

func (err *encodeError) Error() string {
	return fmt.Sprintf("encode error: %s for input type %v", err.msg, err.typ)
}

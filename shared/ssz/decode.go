package ssz

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
)

// Decodable defines the interface for support ssz decoding.
type Decodable interface {
	DecodeSSZ(io.Reader) error
}

// Decode decodes data read from r and output it into the object pointed by pointer val.
func Decode(r io.Reader, val interface{}) error {
	return decode(r, val)
}

func decode(r io.Reader, val interface{}) error {
	if val == nil {
		return newDecodeError("cannot decode into nil", nil)
	}
	rval := reflect.ValueOf(val)
	rtyp := rval.Type()
	// val must be a pointer, otherwise we refuse to decode
	if rtyp.Kind() != reflect.Ptr {
		return newDecodeError("can only decode into pointer target", rtyp)
	}
	if rval.IsNil() {
		return newDecodeError("cannot output to pointer of nil", rtyp)
	}
	sszUtils, err := cachedSSZUtils(rval.Elem().Type())
	if err != nil {
		return newDecodeError(fmt.Sprint(err), rval.Elem().Type())
	}
	if _, err = sszUtils.decoder(r, rval.Elem()); err != nil {
		return newDecodeError(fmt.Sprint(err), rval.Elem().Type())
	}
	return nil
}

func makeDecoder(typ reflect.Type) (dec decoder, err error) {
	kind := typ.Kind()
	switch {
	case kind == reflect.Bool:
		return decodeBool, nil
	case kind == reflect.Uint8:
		return decodeUint8, nil
	case kind == reflect.Uint16:
		return decodeUint16, nil
	case kind == reflect.Uint32:
		return decodeUint32, nil
	case kind == reflect.Int32:
		return decodeUint32, nil
	case kind == reflect.Uint64:
		return decodeUint64, nil
	case kind == reflect.Slice && typ.Elem().Kind() == reflect.Uint8:
		return decodeBytes, nil
	case kind == reflect.Slice:
		return makeSliceDecoder(typ)
	case kind == reflect.Array && typ.Elem().Kind() == reflect.Uint8:
		return decodeByteArray, nil
	case kind == reflect.Array:
		return makeArrayDecoder(typ)
	case kind == reflect.Struct:
		return makeStructDecoder(typ)
	case kind == reflect.Ptr:
		return makePtrDecoder(typ)
	default:
		return nil, fmt.Errorf("type %v is not deserializable", typ)
	}
}

func decodeBool(r io.Reader, val reflect.Value) (uint32, error) {
	b := make([]byte, 1)
	if err := readBytes(r, 1, b); err != nil {
		return 0, err
	}
	v := uint8(b[0])
	if v == 0 {
		val.SetBool(false)
	} else if v == 1 {
		val.SetBool(true)
	} else {
		return 0, fmt.Errorf("expect 0 or 1 for decoding bool but got %d", v)
	}
	return 1, nil
}

func decodeUint8(r io.Reader, val reflect.Value) (uint32, error) {
	b := make([]byte, 1)
	if err := readBytes(r, 1, b); err != nil {
		return 0, err
	}
	val.SetUint(uint64(b[0]))
	return 1, nil
}

func decodeUint16(r io.Reader, val reflect.Value) (uint32, error) {
	b := make([]byte, 2)
	if err := readBytes(r, 2, b); err != nil {
		return 0, err
	}
	val.SetUint(uint64(binary.LittleEndian.Uint16(b)))
	return 2, nil
}

func decodeUint32(r io.Reader, val reflect.Value) (uint32, error) {
	b := make([]byte, 4)
	if err := readBytes(r, 4, b); err != nil {
		return 0, err
	}
	val.SetUint(uint64(binary.LittleEndian.Uint32(b)))
	return 4, nil
}

func decodeUint64(r io.Reader, val reflect.Value) (uint32, error) {
	b := make([]byte, 8)
	if err := readBytes(r, 8, b); err != nil {
		return 0, err
	}
	val.SetUint(uint64(binary.LittleEndian.Uint64(b)))
	return 8, nil
}

func decodeBytes(r io.Reader, val reflect.Value) (uint32, error) {
	sizeEnc := make([]byte, lengthBytes)
	if err := readBytes(r, lengthBytes, sizeEnc); err != nil {
		return 0, err
	}
	size := binary.LittleEndian.Uint32(sizeEnc)

	if size == 0 {
		val.SetBytes([]byte{})
		return lengthBytes, nil
	}

	b := make([]byte, size)
	if err := readBytes(r, int(size), b); err != nil {
		return 0, err
	}
	val.SetBytes(b)
	return lengthBytes + size, nil
}

func decodeByteArray(r io.Reader, val reflect.Value) (uint32, error) {
	sizeEnc := make([]byte, lengthBytes)
	if err := readBytes(r, lengthBytes, sizeEnc); err != nil {
		return 0, err
	}
	size := binary.LittleEndian.Uint32(sizeEnc)

	if size != uint32(val.Len()) {
		return 0, fmt.Errorf("input byte array size (%d) isn't euqal to output array size (%d)", size, val.Len())
	}

	slice := val.Slice(0, val.Len()).Interface().([]byte)
	if err := readBytes(r, int(size), slice); err != nil {
		return 0, err
	}
	return lengthBytes + size, nil
}

func makeSliceDecoder(typ reflect.Type) (decoder, error) {
	elemType := typ.Elem()
	elemSSZUtils, err := cachedSSZUtilsNoAcquireLock(elemType)
	if err != nil {
		return nil, err
	}
	decoder := func(r io.Reader, val reflect.Value) (uint32, error) {
		sizeEnc := make([]byte, lengthBytes)
		if err := readBytes(r, lengthBytes, sizeEnc); err != nil {
			return 0, fmt.Errorf("failed to decode header of slice: %v", err)
		}
		size := binary.LittleEndian.Uint32(sizeEnc)

		if size == 0 {
			// We prefer decode into nil, not empty slice
			return lengthBytes, nil
		}

		for i, decodeSize := 0, uint32(0); decodeSize < size; i++ {
			// Grow slice's capacity if necessary
			if i >= val.Cap() {
				newCap := val.Cap() * 2
				// Skip initial small growth
				if newCap < 4 {
					newCap = 4
				}
				newVal := reflect.MakeSlice(val.Type(), val.Len(), newCap)
				reflect.Copy(newVal, val)
				val.Set(newVal)
			}

			// Add place holder for new element
			if i >= val.Len() {
				val.SetLen(i + 1)
			}

			// Decode and write into the new element
			elemDecodeSize, err := elemSSZUtils.decoder(r, val.Index(i))
			if err != nil {
				return 0, fmt.Errorf("failed to decode element of slice: %v", err)
			}
			decodeSize += elemDecodeSize
		}
		return lengthBytes + size, nil
	}
	return decoder, nil
}

func makeArrayDecoder(typ reflect.Type) (decoder, error) {
	elemType := typ.Elem()
	elemSSZUtils, err := cachedSSZUtilsNoAcquireLock(elemType)
	if err != nil {
		return nil, err
	}
	decoder := func(r io.Reader, val reflect.Value) (uint32, error) {
		sizeEnc := make([]byte, lengthBytes)
		if err := readBytes(r, lengthBytes, sizeEnc); err != nil {
			return 0, fmt.Errorf("failed to decode header of slice: %v", err)
		}
		size := binary.LittleEndian.Uint32(sizeEnc)

		i, decodeSize := 0, uint32(0)
		for ; i < val.Len() && decodeSize < size; i++ {
			elemDecodeSize, err := elemSSZUtils.decoder(r, val.Index(i))
			if err != nil {
				return 0, fmt.Errorf("failed to decode element of slice: %v", err)
			}
			decodeSize += elemDecodeSize
		}
		if i < val.Len() {
			return 0, errors.New("input is too short")
		}
		if decodeSize < size {
			return 0, errors.New("input is too long")
		}
		return lengthBytes + size, nil
	}
	return decoder, nil
}

func makeStructDecoder(typ reflect.Type) (decoder, error) {
	fields, err := structFields(typ)
	if err != nil {
		return nil, err
	}
	decoder := func(r io.Reader, val reflect.Value) (uint32, error) {
		sizeEnc := make([]byte, lengthBytes)
		if err := readBytes(r, lengthBytes, sizeEnc); err != nil {
			return 0, fmt.Errorf("failed to decode header of struct: %v", err)
		}
		size := binary.LittleEndian.Uint32(sizeEnc)

		if size == 0 {
			return lengthBytes, nil
		}

		i, decodeSize := 0, uint32(0)
		for ; i < len(fields) && decodeSize < size; i++ {
			f := fields[i]
			fieldDecodeSize, err := f.sszUtils.decoder(r, val.Field(f.index))
			if err != nil {
				return 0, fmt.Errorf("failed to decode field of slice: %v", err)
			}
			decodeSize += fieldDecodeSize
		}
		if i < len(fields) {
			return 0, errors.New("input is too short")
		}
		if decodeSize < size {
			return 0, errors.New("input is too long")
		}
		return lengthBytes + size, nil
	}
	return decoder, nil
}

func makePtrDecoder(typ reflect.Type) (decoder, error) {
	elemType := typ.Elem()
	elemSSZUtils, err := cachedSSZUtilsNoAcquireLock(elemType)
	if err != nil {
		return nil, err
	}

	// After considered the use case in Prysm, we've decided that:
	// - We assume we will only encode/decode pointer of array, slice or struct.
	// - The encoding for nil pointer shall be 0x00000000.

	decoder := func(r io.Reader, val reflect.Value) (uint32, error) {
		newVal := reflect.New(elemType)
		elemDecodeSize, err := elemSSZUtils.decoder(r, newVal.Elem())
		if err != nil {
			return 0, fmt.Errorf("failed to decode to object pointed by pointer: %v", err)
		}
		if elemDecodeSize > lengthBytes {
			val.Set(newVal)
		} // Else we leave val to its default value which is nil.
		return elemDecodeSize, nil
	}
	return decoder, nil
}

func readBytes(r io.Reader, size int, b []byte) error {
	if size != len(b) {
		return fmt.Errorf("output buffer size is %d while expected read size is %d", len(b), size)
	}
	readLen, err := r.Read(b)
	if readLen != size {
		return fmt.Errorf("can only read %d bytes while expected to read %d bytes", readLen, size)
	}
	if err != nil {
		return fmt.Errorf("failed to read from input: %v", err)
	}
	return nil
}

// decodeError is what gets reported to the decoder user in error case.
type decodeError struct {
	msg string
	typ reflect.Type
}

func newDecodeError(msg string, typ reflect.Type) *decodeError {
	return &decodeError{msg, typ}
}

func (err *decodeError) Error() string {
	return fmt.Sprintf("decode error: %s for output type %v", err.msg, err.typ)
}

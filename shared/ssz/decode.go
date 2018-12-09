package ssz

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

// TODOs for this PR:
// - Review all existing error handling
// - Add error handing when decoding invalid input stream
// - Create error types so we have fewer error texts all over the code
// - Use constant to replace hard coding of number 4

// TODOs for later PR:
// - Add support for more types

// Decode TODO add comments
func Decode(r io.Reader, val interface{}) error {
	_, err := decode(r, val)
	return err
}

func decode(r io.Reader, val interface{}) (uint32, error) {
	if val == nil {
		return 0, newDecodeError("cannot decode into nil", nil)
	}
	rval := reflect.ValueOf(val)
	rtyp := rval.Type()
	if rtyp.Kind() != reflect.Ptr {
		return 0, newDecodeError("can only decode into pointer target", rtyp)
	}
	if rval.IsNil() {
		return 0, newDecodeError("cannot output to pointer of nil", rtyp)
	}
	encDec, err := getEncoderDecoderForType(rval.Elem().Type())
	if err != nil {
		return 0, newDecodeError(fmt.Sprint(err), rval.Elem().Type())
	}
	return encDec.decoder(r, rval.Elem())
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
	case kind == reflect.Slice && typ.Elem().Kind() == reflect.Uint8:
		return decodeBytes, nil
	case kind == reflect.Slice:
		return makeSliceDecoder(typ)
	case kind == reflect.Struct:
		return makeStructDecoder(typ)
	default:
		return nil, fmt.Errorf("type %v is not deserializable", typ)
	}
}

func decodeBool(r io.Reader, val reflect.Value) (uint32, error) {
	b := make([]byte, 1)
	if err := readBytes(r, 1, b); err != nil {
		return 0, newDecodeError(fmt.Sprint(err), val.Type())
	}
	v := uint8(b[0])
	if v == 0 {
		val.SetBool(false)
	} else if v == 1 {
		val.SetBool(true)
	} else {
		return 0, newDecodeError("expect 0 or 1 for decoding bool", val.Type())
	}
	return 1, nil
}

func decodeUint8(r io.Reader, val reflect.Value) (uint32, error) {
	b := make([]byte, 1)
	if err := readBytes(r, 1, b); err != nil {
		return 0, newDecodeError(fmt.Sprint(err), val.Type())
	}
	val.SetUint(uint64(b[0]))
	return 1, nil
}

func decodeUint16(r io.Reader, val reflect.Value) (uint32, error) {
	b := make([]byte, 2)
	if err := readBytes(r, 2, b); err != nil {
		return 0, newDecodeError(fmt.Sprint(err), val.Type())
	}
	val.SetUint(uint64(binary.BigEndian.Uint16(b)))
	return 2, nil
}

func decodeBytes(r io.Reader, val reflect.Value) (uint32, error) {
	sizeEnc := make([]byte, 4)
	if err := readBytes(r, 4, sizeEnc); err != nil {
		return 0, newDecodeError(fmt.Sprint(err), val.Type())
	}
	size := binary.BigEndian.Uint32(sizeEnc)

	if size == 0 {
		val.SetBytes([]byte{})
		return 4, nil
	}

	b := make([]byte, size)
	if err := readBytes(r, int(size), b); err != nil {
		return 0, newDecodeError(fmt.Sprint(err), val.Type())
	}
	val.SetBytes(b)
	return 4 + size, nil
}

func makeSliceDecoder(typ reflect.Type) (decoder, error) {
	elemType := typ.Elem()
	elemEncoderDecoder, err := getEncoderDecoderForType(elemType)
	if err != nil {
		return nil, newDecodeError(fmt.Sprint(err), typ)
	}
	decoder := func(r io.Reader, val reflect.Value) (uint32, error) {
		sizeEnc := make([]byte, 4)
		if err := readBytes(r, 4, sizeEnc); err != nil {
			return 0, newDecodeError(fmt.Sprintf("failed to decode header of slice: %v", err), typ)
		}
		size := binary.BigEndian.Uint32(sizeEnc)

		if size == 0 {
			// We prefer decode into nil, not empty slice
			return 4, nil
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
			elemDecodeSize, err := elemEncoderDecoder.decoder(r, val.Index(i))
			if err != nil {
				return 0, newDecodeError(fmt.Sprintf("failed to decode element of slice: %v", err), typ)
			}
			decodeSize += elemDecodeSize
		}
		return 4 + size, nil
	}
	return decoder, nil
}

func makeStructDecoder(typ reflect.Type) (decoder, error) {
	fields, err := sortedStructFields(typ)
	if err != nil {
		return nil, newDecodeError(fmt.Sprint(err), typ)
	}
	decoder := func(r io.Reader, val reflect.Value) (uint32, error) {
		sizeEnc := make([]byte, 4)
		if err := readBytes(r, 4, sizeEnc); err != nil {
			return 0, newDecodeError(fmt.Sprintf("failed to decode header of struct: %v", err), typ)
		}
		size := binary.BigEndian.Uint32(sizeEnc)

		if size == 0 {
			return 4, nil
		}

		for i, decodeSize := 0, uint32(0); i < len(fields); i++ {
			if decodeSize >= size {
				return 0, newDecodeError("not enough input data to decode into specified struct", typ)
			}
			f := fields[i]
			fieldDecodeSize, err := f.encDec.decoder(r, val.Field(f.index))
			if err != nil {
				return 0, newDecodeError(fmt.Sprintf("failed to decode field of slice: %v", err), typ)
			}
			decodeSize += fieldDecodeSize
		}
		return 4 + size, nil
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

type decodeError struct {
	msg string
	typ reflect.Type
}

func newDecodeError(msg string, typ reflect.Type) *decodeError {
	return &decodeError{msg, typ}
}

func (err *decodeError) Error() string {
	return fmt.Sprintf("ssz: decode error: %s for output type %v", err.msg, err.typ)
}

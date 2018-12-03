package ssz

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

// TODOs:
// - Review all error handling
// - Create error types so we have fewer error texts all over the code

func Decode(r io.Reader, val interface{}) error {
	return decode(r, val)
}

func decode(r io.Reader, val interface{}) error {
	if val == nil {
		return fmt.Errorf("ssz: cannot output to nil")
	}
	rval := reflect.ValueOf(val)
	rtyp := rval.Type()
	if rtyp.Kind() != reflect.Ptr {
		return fmt.Errorf("ssz: can only output to ptr")
	}
	if rval.IsNil() {
		return fmt.Errorf("ssz: cannot output to nil")
	}
	encDec, err := getEncoderDecoderForType(rval.Elem().Type())
	if err != nil {
		return err
	}
	return encDec.decoder(r, rval.Elem())
}

func makeDecoder(typ reflect.Type) (dec decoder, err error) {
	kind := typ.Kind()
	switch {
	case kind == reflect.Uint8:
		return decodeUint8, nil
	case kind == reflect.Uint16:
		return decodeUint16, nil
	case kind == reflect.Slice && typ.Elem().Kind() == reflect.Uint8:
		return decodeBytes, nil
	default:
		return nil, fmt.Errorf("ssz: type %v is not deserializable", typ)
	}
}

func decodeUint8(r io.Reader, val reflect.Value) error {
	b := make([]byte, 1)
	if err := readBytes(r, 1, b); err != nil {
		return fmt.Errorf("failed to decode uint8: %v", err)
	}
	val.SetUint(uint64(b[0]))
	return nil
}

func decodeUint16(r io.Reader, val reflect.Value) error {
	b := make([]byte, 2)
	if err := readBytes(r, 2, b); err != nil {
		return fmt.Errorf("failed to decode uint16: %v", err)
	}
	val.SetUint(uint64(binary.BigEndian.Uint16(b)))
	return nil
}

func decodeBytes(r io.Reader, val reflect.Value) error {
	lengthEnc := make([]byte, 4)
	if err := readBytes(r, 4, lengthEnc); err != nil {
		return fmt.Errorf("failed to decode header of bytes: %v", err)
	}
	length := binary.BigEndian.Uint32(lengthEnc)
	fmt.Println(length)

	b := make([]byte, length)
	if err := readBytes(r, int(length), b); err != nil {
		return fmt.Errorf("failed to decode bytes: %v", err)
	}
	val.SetBytes(b)
	return nil
}

func readBytes(r io.Reader, length int, b []byte) error {
	if length != len(b) {
		return fmt.Errorf("output buffer size is %d while expected read length is %d", len(b), length)
	}
	readLen, err := r.Read(b)
	if readLen != length {
		return fmt.Errorf("can only read %d bytes while expected to read %d bytes", readLen, length)
	}
	if err != nil {
		return fmt.Errorf("failed to read from input: %v", err)
	}
	return nil
}

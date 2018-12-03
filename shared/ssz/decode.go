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
	default:
		return nil, fmt.Errorf("ssz: type %v is not deserializable", typ)
	}
}

func decodeUint8(r io.Reader, val reflect.Value) error {
	b := make([]byte, 1)
	readLen, err := r.Read(b)
	if readLen != 1 {
		return fmt.Errorf("ssz: read %d bytes when trying to read 1 bytes from decoding input", readLen)
	}
	if err != nil {
		return fmt.Errorf("ssz: failed to read uint8 from decoding input %v", err)
	}
	val.SetUint(uint64(b[0]))
	return nil
}

func decodeUint16(r io.Reader, val reflect.Value) error {
	b := make([]byte, 2)
	readLen, err := r.Read(b)
	if readLen != 2 {
		return fmt.Errorf("ssz: read %d bytes when trying to read 2 bytes from decoding input", readLen)
	}
	if err != nil {
		return fmt.Errorf("ssz: failed to read uint8 from decoding input %v", err)
	}
	val.SetUint(uint64(binary.BigEndian.Uint16(b[:])))
	return nil
}

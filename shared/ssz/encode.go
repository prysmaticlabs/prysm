package ssz

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

// TODOs for this PR:
// - Review all error handling
// - Add check for input sizes!

func Encode(w io.Writer, val interface{}) error {
	eb := &encbuf{}
	if err := eb.encode(val); err != nil {
		return err
	}
	return eb.toWriter(w)
}

type encbuf struct {
	str []byte
}

func (w *encbuf) encode(val interface{}) error {
	rval := reflect.ValueOf(val)
	encDec, err := getEncoderDecoderForType(rval.Type())
	if err != nil {
		return err
	}
	return encDec.encoder(rval, w)
}

func (w *encbuf) toWriter(out io.Writer) error {
	if _, err := out.Write(w.str); err != nil {
		return err
	}
	return nil
}

func makeEncoder(typ reflect.Type) (encoder, error) {
	kind := typ.Kind()
	switch {
	case kind == reflect.Uint8:
		return encodeUint8, nil
	case kind == reflect.Uint16:
		return encodeUint16, nil
	case kind == reflect.Slice && typ.Elem().Kind() == reflect.Uint8:
		return encodeBytes, nil
	default:
		return nil, fmt.Errorf("ssz: type %v is not serializable", typ)
	}
}

func encodeUint8(val reflect.Value, w *encbuf) error {
	v := val.Uint()
	w.str = append(w.str, uint8(v))
	return nil
}

func encodeUint16(val reflect.Value, w *encbuf) error {
	v := val.Uint()
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	w.str = append(w.str, b...)
	return nil
}

func encodeBytes(val reflect.Value, w *encbuf) error {
	b := val.Bytes()
	lengthEnc := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthEnc, uint32(len(b)))
	w.str = append(w.str, lengthEnc...)
	w.str = append(w.str, val.Bytes()...)
	return nil
}


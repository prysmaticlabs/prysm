package ssz

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

func Encode(w io.Writer, val interface{}) error {
	// TODO: review and complete
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
	out.Write(w.str)
	return nil
}

func makeEncoder(typ reflect.Type) (encoder, error) {
	kind := typ.Kind()
	switch {
	case kind == reflect.Uint8:
		return encodeUint8, nil
	case kind == reflect.Uint16:
		return encodeUint16, nil
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

package ssz

import (
	"io"
	"reflect"
)

// TODOs for this PR:
// - Implement encoder/decoder caches for types

type encoder func(reflect.Value, *encbuf) error
type decoder func(io.Reader, reflect.Value) error

type encoderDecoder struct {
	encoder
	decoder
}

func getEncoderDecoderForType(typ reflect.Type) (encDec *encoderDecoder, err error) {
	encDec = new(encoderDecoder)
	if encDec.encoder, err = makeEncoder(typ); err != nil {
		return nil, err
	}
	if encDec.decoder, err = makeDecoder(typ); err != nil {
		return nil, err
	}
	return encDec, nil
}

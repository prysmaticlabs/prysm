package ssz

import (
	"fmt"
	"io"
	"reflect"
	"sort"
)

// TODOs for this PR:
// - Implement encoder/decoder caches for types to reduce the encoder/decoder lookup overhead

type encoder func(reflect.Value, *encbuf) error

// Notice: We are not exactly following the spec which requires a decoder to return new index in the input buffer.
// Our io.Reader is already capable of tracking its latest read location, so we decide to return the decoded byte size
// instead. This makes our implementation look cleaner.
type decoder func(io.Reader, reflect.Value) (uint32, error)

type encoderDecoder struct {
	encoder
	decoder
}

// TODO: We can let this function return (encDec *encoderDecoder, encodeTargetSize uint32, err error)
// if we want to know the encode output size before the actual encoding
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

type field struct {
	index  int
	name   string
	encDec *encoderDecoder
}

func sortedStructFields(typ reflect.Type) (fields []field, err error) {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		encDec, err := getEncoderDecoderForType(f.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get encoder/decoder: %v", err)
		}
		name := f.Name
		fields = append(fields, field{i, name, encDec})
	}
	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].name < fields[j].name
	})
	return fields, nil
}

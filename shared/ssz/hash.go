package ssz

import (
	"bytes"
	"fmt"
	"reflect"
)

// TODO: cachedSSZUtils should be renamed into cachedSSZUtils

type Hashable interface {
	HashSSZ() ([32]byte, error)
}

func Hash(val interface{}) ([32]byte, error) {
	if val == nil {
		return [32]byte{}, newHashError("nil is not supported", nil)
	}
	rval := reflect.ValueOf(val)
	sszUtils, err := cachedSSZUtils(rval.Type())
	if err != nil {
		return [32]byte{}, newHashError(fmt.Sprint(err), rval.Type())
	}
	output, err := sszUtils.hasher(rval)
	if err != nil {
		return [32]byte{}, newHashError(fmt.Sprint(err), rval.Type())
	}
	return output, nil
}

type hashError struct {
	msg string
	typ reflect.Type
}

func (err *hashError) Error() string {
	return fmt.Sprintf("ssz hash error: %s for input type %v", err.msg, err.typ)
}

func newHashError(msg string, typ reflect.Type) *hashError {
	return &hashError{msg, typ}
}

func makeHasher(typ reflect.Type) (hasher, error) {
	kind := typ.Kind()
	switch {
	case kind == reflect.Bool ||
		kind == reflect.Uint8 ||
		kind == reflect.Uint16 ||
		kind == reflect.Uint32 ||
		kind == reflect.Uint64:
		return hashPrimitive, nil
	default:
		return nil, fmt.Errorf("type %v is not hashable", typ)
	}
}

func hashPrimitive(val reflect.Value) ([32]byte, error) {
	utils, err := cachedSSZUtilsNoAcquireLock(val.Type())
	buf := &encbuf{}
	if err = utils.encoder(val, buf); err != nil {
		return [32]byte{}, fmt.Errorf("failed to encode: %v", err)
	}
	writer := new(bytes.Buffer)
	if err = buf.toWriter(writer); err != nil {
		return [32]byte{}, fmt.Errorf("failed to output to writer: %v", err)
	}
	b := writer.Bytes()
	var output [32]byte
	copy(output[:], b)
	return output, nil
}

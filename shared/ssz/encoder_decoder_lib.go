package ssz

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
)

type encoder func(reflect.Value, *encbuf) error

// Notice: we are not exactly following the spec which requires a decoder to return new index in the input buffer.
// Our io.Reader is already capable of tracking its latest read location, so we decide to return the decoded byte size
// instead. This makes our implementation look cleaner.
type decoder func(io.Reader, reflect.Value) (uint32, error)

type encodeSizer func(reflect.Value) (uint32, error)

type encoderDecoder struct {
	encoder
	encodeSizer
	decoder
}

var (
	encoderDecoderCacheMutex sync.RWMutex
	encoderDecoderCache      = make(map[reflect.Type]*encoderDecoder)
)

// Get cached encoder, encodeSizer and decoder implementation for a specified type.
// With a cache we can achieve O(1) amortized time overhead for creating encoder, encodeSizer and decoder.
func cachedEncoderDecoder(typ reflect.Type) (*encoderDecoder, error) {
	encoderDecoderCacheMutex.RLock()
	encDec := encoderDecoderCache[typ]
	encoderDecoderCacheMutex.RUnlock()
	if encDec != nil {
		return encDec, nil
	}

	// If not found in cache, will get a new one and put it into the cache
	encoderDecoderCacheMutex.Lock()
	defer encoderDecoderCacheMutex.Unlock()
	return cachedEncoderDecoderNoAcquireLock(typ)
}

// This version is used when the caller is already holding the rw lock for encoderDecoderCache.
// It doesn't acquire new rw lock so it's free to recursively call itself without getting into
// a deadlock situation.
func cachedEncoderDecoderNoAcquireLock(typ reflect.Type) (*encoderDecoder, error) {
	// Check again in case other goroutine has just acquired the lock
	// and already updated the cache
	encDec := encoderDecoderCache[typ]
	if encDec != nil {
		return encDec, nil
	}
	// Put a dummy value into the cache before generating.
	// If the generator tries to lookup the type of itself,
	// it will get the dummy value and won't call recursively forever.
	encoderDecoderCache[typ] = new(encoderDecoder)
	encDec, err := generateEncoderDecoderForType(typ)
	if err != nil {
		// Don't forget to remove the dummy key when fail
		delete(encoderDecoderCache, typ)
		return nil, err
	}
	// Overwrite the dummy value with real value
	*encoderDecoderCache[typ] = *encDec
	return encoderDecoderCache[typ], nil
}

func generateEncoderDecoderForType(typ reflect.Type) (encDec *encoderDecoder, err error) {
	encDec = new(encoderDecoder)
	if encDec.encoder, encDec.encodeSizer, err = makeEncoder(typ); err != nil {
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

func structFields(typ reflect.Type) (fields []field, err error) {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if strings.Contains(f.Name, "XXX") {
			continue
		}
		encDec, err := cachedEncoderDecoderNoAcquireLock(f.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get encoder/decoder: %v", err)
		}
		name := f.Name
		fields = append(fields, field{i, name, encDec})
	}
	return fields, nil
}

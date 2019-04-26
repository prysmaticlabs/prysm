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

type hasher func(reflect.Value) ([]byte, error)

type sszUtils struct {
	encoder
	encodeSizer
	decoder
	hasher
}

var (
	sszUtilsCacheMutex sync.RWMutex
	sszUtilsCache      = make(map[reflect.Type]*sszUtils)
	hashCache          = newHashCache()
)

// Get cached encoder, encodeSizer and decoder implementation for a specified type.
// With a cache we can achieve O(1) amortized time overhead for creating encoder, encodeSizer and decoder.
func cachedSSZUtils(typ reflect.Type) (*sszUtils, error) {
	sszUtilsCacheMutex.RLock()
	utils := sszUtilsCache[typ]
	sszUtilsCacheMutex.RUnlock()
	if utils != nil {
		return utils, nil
	}

	// If not found in cache, will get a new one and put it into the cache
	sszUtilsCacheMutex.Lock()
	defer sszUtilsCacheMutex.Unlock()
	return cachedSSZUtilsNoAcquireLock(typ)
}

// This version is used when the caller is already holding the rw lock for sszUtilsCache.
// It doesn't acquire new rw lock so it's free to recursively call itself without getting into
// a deadlock situation.
//
// Make sure you are
func cachedSSZUtilsNoAcquireLock(typ reflect.Type) (*sszUtils, error) {
	// Check again in case other goroutine has just acquired the lock
	// and already updated the cache
	utils := sszUtilsCache[typ]
	if utils != nil {
		return utils, nil
	}
	// Put a dummy value into the cache before generating.
	// If the generator tries to lookup the type of itself,
	// it will get the dummy value and won't call recursively forever.
	sszUtilsCache[typ] = new(sszUtils)
	utils, err := generateSSZUtilsForType(typ)
	if err != nil {
		// Don't forget to remove the dummy key when fail
		delete(sszUtilsCache, typ)
		return nil, err
	}
	// Overwrite the dummy value with real value
	*sszUtilsCache[typ] = *utils
	return sszUtilsCache[typ], nil
}

func generateSSZUtilsForType(typ reflect.Type) (utils *sszUtils, err error) {
	utils = new(sszUtils)
	if utils.encoder, utils.encodeSizer, err = makeEncoder(typ); err != nil {
		return nil, err
	}
	if utils.decoder, err = makeDecoder(typ); err != nil {
		return nil, err
	}
	if utils.hasher, err = makeHasher(typ); err != nil {
		return nil, err
	}
	return utils, nil
}

type field struct {
	index    int
	name     string
	sszUtils *sszUtils
}

func structFields(typ reflect.Type) (fields []field, err error) {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if strings.Contains(f.Name, "XXX") {
			continue
		}
		utils, err := cachedSSZUtilsNoAcquireLock(f.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get ssz utils: %v", err)
		}
		name := f.Name
		fields = append(fields, field{i, name, utils})
	}
	return fields, nil
}

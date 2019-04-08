package ssz

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

const hashLengthBytes = 32
const sszChunkSize = 128

var useCache bool

// Hashable defines the interface for supporting tree-hash function.
type Hashable interface {
	TreeHashSSZ() ([32]byte, error)
}

// TreeHash calculates tree-hash result for input value.
func TreeHash(val interface{}) ([32]byte, error) {
	if val == nil {
		return [32]byte{}, newHashError("untyped nil is not supported", nil)
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
	// Right-pad with 0 to make 32 bytes long, if necessary
	paddedOutput := bytesutil.ToBytes32(output)
	return paddedOutput, nil
}

type hashError struct {
	msg string
	typ reflect.Type
}

func (err *hashError) Error() string {
	return fmt.Sprintf("hash error: %s for input type %v", err.msg, err.typ)
}

func newHashError(msg string, typ reflect.Type) *hashError {
	return &hashError{msg, typ}
}

func makeHasher(typ reflect.Type) (hasher, error) {
	useCache = featureconfig.FeatureConfig().CacheTreeHash
	kind := typ.Kind()
	switch {
	case kind == reflect.Bool ||
		kind == reflect.Uint8 ||
		kind == reflect.Uint16 ||
		kind == reflect.Uint32 ||
		kind == reflect.Uint64 ||
		kind == reflect.Int32:
		return getEncoding, nil
	case kind == reflect.Slice && typ.Elem().Kind() == reflect.Uint8 ||
		kind == reflect.Array && typ.Elem().Kind() == reflect.Uint8:
		return hashedEncoding, nil
	case kind == reflect.Slice || kind == reflect.Array:
		if useCache {
			return makeSliceHasherCache(typ)
		}
		return makeSliceHasher(typ)
	case kind == reflect.Struct:
		if useCache {
			return makeStructHasherCache(typ)
		}
		return makeStructHasher(typ)
	case kind == reflect.Ptr:
		return makePtrHasher(typ)
	default:
		return nil, fmt.Errorf("type %v is not hashable", typ)
	}
}

func getEncoding(val reflect.Value) ([]byte, error) {
	utils, err := cachedSSZUtilsNoAcquireLock(val.Type())
	if err != nil {
		return nil, err
	}
	buf := &encbuf{}
	if err = utils.encoder(val, buf); err != nil {
		return nil, err
	}
	writer := new(bytes.Buffer)
	if err = buf.toWriter(writer); err != nil {
		return nil, err
	}
	return writer.Bytes(), nil
}

func hashedEncoding(val reflect.Value) ([]byte, error) {
	encoding, err := getEncoding(val)
	if err != nil {
		return nil, err
	}
	output := hashutil.Hash(encoding)
	return output[:], nil
}

func makeSliceHasher(typ reflect.Type) (hasher, error) {
	elemSSZUtils, err := cachedSSZUtilsNoAcquireLock(typ.Elem())
	if err != nil {
		return nil, fmt.Errorf("failed to get ssz utils: %v", err)
	}
	hasher := func(val reflect.Value) ([]byte, error) {
		var elemHashList [][]byte
		for i := 0; i < val.Len(); i++ {
			elemHash, err := elemSSZUtils.hasher(val.Index(i))
			if err != nil {
				return nil, fmt.Errorf("failed to hash element of slice/array: %v", err)
			}
			elemHashList = append(elemHashList, elemHash)
		}
		output, err := merkleHash(elemHashList)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate merkle hash of element hash list: %v", err)
		}
		return output, nil
	}
	return hasher, nil
}

func makeStructHasher(typ reflect.Type) (hasher, error) {
	fields, err := structFields(typ)
	if err != nil {
		return nil, err
	}
	hasher := func(val reflect.Value) ([]byte, error) {
		concatElemHash := make([]byte, 0)
		for _, f := range fields {
			elemHash, err := f.sszUtils.hasher(val.Field(f.index))
			if err != nil {
				return nil, fmt.Errorf("failed to hash field of struct: %v", err)
			}
			concatElemHash = append(concatElemHash, elemHash...)
		}
		result := hashutil.Hash(concatElemHash)
		return result[:], nil
	}
	return hasher, nil
}

func makePtrHasher(typ reflect.Type) (hasher, error) {
	elemSSZUtils, err := cachedSSZUtilsNoAcquireLock(typ.Elem())
	if err != nil {
		return nil, err
	}

	// TODO(1461): The tree-hash of nil pointer isn't defined in the spec.
	// After considered the use case in Prysm, we've decided that:
	// - We assume we will only tree-hash pointer of array, slice or struct.
	// - The tree-hash for nil pointer shall be 0x00000000.

	hasher := func(val reflect.Value) ([]byte, error) {
		if val.IsNil() {
			return hashedEncoding(val)
		}
		return elemSSZUtils.hasher(val.Elem())
	}
	return hasher, nil
}

// merkelHash implements a merkle-tree style hash algorithm.
//
// Please refer to the official spec for details:
// https://github.com/ethereum/eth2.0-specs/blob/master/specs/simple-serialize.md#tree-hash
//
// The overall idea is:
// 1. Create a bunch of bytes chunk (each has a size of sszChunkSize) from the input hash list.
// 2. Treat each bytes chunk as the leaf of a binary tree.
// 3. For every pair of leaves, we set their parent's value using the hash value of the concatenation of the two leaves.
//    The original two leaves are then removed.
// 4. Keep doing step 3 until there's only one node left in the tree (the root).
// 5. Return the hash of the concatenation of the root and the data length encoding.
//
// Time complexity is O(n) given input list of size n.
func merkleHash(list [][]byte) ([]byte, error) {
	// Assume len(list) < 2^64
	dataLenEnc := make([]byte, hashLengthBytes)
	binary.LittleEndian.PutUint64(dataLenEnc, uint64(len(list)))

	var chunkz [][]byte
	emptyChunk := make([]byte, sszChunkSize)

	if len(list) == 0 {
		chunkz = make([][]byte, 1)
		chunkz[0] = emptyChunk
	} else if len(list[0]) < sszChunkSize {

		itemsPerChunk := sszChunkSize / len(list[0])
		chunkz = make([][]byte, 0)
		for i := 0; i < len(list); i += itemsPerChunk {
			chunk := make([]byte, 0)
			j := i + itemsPerChunk
			if j > len(list) {
				j = len(list)
			}
			// Every chunk should have sszChunkSize bytes except that the last one could have less bytes
			for _, elemHash := range list[i:j] {
				chunk = append(chunk, elemHash...)
			}
			chunkz = append(chunkz, chunk)
		}
	} else {
		chunkz = list
	}

	for len(chunkz) > 1 {
		if len(chunkz)%2 == 1 {
			chunkz = append(chunkz, emptyChunk)
		}
		hashedChunkz := make([][]byte, 0)
		for i := 0; i < len(chunkz); i += 2 {
			hashedChunk := hashutil.Hash(append(chunkz[i], chunkz[i+1]...))
			hashedChunkz = append(hashedChunkz, hashedChunk[:])
		}
		chunkz = hashedChunkz
	}

	result := hashutil.Hash(append(chunkz[0], dataLenEnc...))
	return result[:], nil
}

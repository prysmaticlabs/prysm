package ssz

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

const hashLengthBytes = 32
const sszChunkSize = 128

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
	var paddedOutput [32]byte
	copy(paddedOutput[:], output)
	return paddedOutput, nil
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
		return getEncoding, nil
	case kind == reflect.Slice && typ.Elem().Kind() == reflect.Uint8 ||
		kind == reflect.Array && typ.Elem().Kind() == reflect.Uint8:
		return hashedEncoding, nil
	case kind == reflect.Slice || kind == reflect.Array:
		return makeSliceHasher(typ)
	default:
		return nil, fmt.Errorf("type %v is not hashable", typ)
	}
}

func getEncoding(val reflect.Value) ([]byte, error) {
	utils, err := cachedSSZUtilsNoAcquireLock(val.Type())
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

//func paddedEncoding(val reflect.Value) ([32]byte, error) {
//	encoding, err := getEncoding(val)
//	if err != nil {
//		return [32]byte{}, err
//	}
//	var output [32]byte
//	copy(output[:], encoding)
//	return output, nil
//}

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

func merkleHash(list [][]byte) ([]byte, error) {
	// Assume len(list) < 2^64
	dataLenEnc := make([]byte, hashLengthBytes)
	binary.BigEndian.PutUint64(dataLenEnc[hashLengthBytes-8:], uint64(len(list)))

	//fmt.Printf("datalen enc: %v", dataLenEnc)

	var chunkz [][]byte
	emptyChunk := make([]byte, sszChunkSize)

	if len(list) == 0 {
		chunkz = make([][]byte, 1)
		chunkz[0] = emptyChunk
	} else if len(list[0]) < sszChunkSize {
		//fmt.Printf("elem size: %d\n", len(list[0]))
		if sszChunkSize%len(list[0]) != 0 {
			return nil, fmt.Errorf("element hash size needs to be factor of %d", sszChunkSize)
		}
		itemsPerChunk := sszChunkSize / len(list[0])
		//fmt.Printf("items per chunk: %d\n", itemsPerChunk)
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
			//fmt.Printf("chunk %d has size: %d\n", i, len(chunk))
			chunkz = append(chunkz, chunk)
			//fmt.Printf("chunk %d: %v\n", i, chunk)
		}
		//fmt.Printf("got %d chunks in total\n", len(chunkz))
	} else {
		chunkz = list
	}

	//fmt.Printf("chunks: %v\n", chunkz)
	//fmt.Println(len(chunkz[0]))

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
	//fmt.Printf("merkle hashed chunkz: %v\n", chunkz)

	result := hashutil.Hash(append(chunkz[0], dataLenEnc...))
	//fmt.Printf("final result: %x\n", result)

	return result[:], nil
}

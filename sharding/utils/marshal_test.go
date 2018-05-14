package utils

import (
	"math/rand"
	"reflect"
	"testing"
)

func buildrawblob(size int64) []RawBlob {
	tempbody := make([]RawBlob, size)
	for i := int64(0); i < size; i++ {
		var rawblob RawBlob
		rawblob.data = buildblob(size)
		flagset := byte(rand.Int()) >> 7
		if flagset == byte(1) {
			rawblob.flags.skipEvmExecution = true

		}

		tempbody[i] = rawblob

	}
	return tempbody

}

func buildblob(size int64) []byte {

	tempbody := make([]byte, size)
	for i := int64(0); i < size; i++ {
		tempbody[i] = byte(rand.Int())

	}

	return tempbody

}

/*
Might be required in the future for part 2 of serialization

func TestConvertInterface(t *testing.T) {
	slice := []interface{}{0, 1, 2, 3, 4, 5}
	convertedValue, err := ConvertInterface(slice, reflect.Slice)
	if err != nil {
		t.Fatalf("Error: %v %v", err, convertedValue)
	}

} */
func TestSize(t *testing.T) {
	size := int64(8)
	blob := buildrawblob(size)
	chunksafterSerialize := size / chunkDataSize
	terminalchunk := size % chunkDataSize
	if terminalchunk != 0 {
		chunksafterSerialize = chunksafterSerialize + 1
	}
	chunksafterSerialize = chunksafterSerialize * size
	sizeafterSerialize := chunksafterSerialize * chunkSize
	serializedblob, err := Serialize(blob)
	if err != nil {
		t.Fatalf("Error Serializing blob:%v %v", err, serializedblob)
	}

	if int64(len(serializedblob)) != sizeafterSerialize {

		t.Fatalf("Error Serializing blobs the lengths are not the same: %v , %v", int64(len(serializedblob)), sizeafterSerialize)

	}

}
func TestSerializeAndDeserializeblob(t *testing.T) {

	blob := buildrawblob(330)

	serializedblob, err := Serialize(blob)

	if err != nil {
		t.Fatalf("Error Serializing blob:%v %v", err, serializedblob)
	}
	raw, err2 := Deserialize(serializedblob)
	if err2 != nil {
		t.Fatalf("Error Serializing blob:%v due to %v", raw, err2)
	}

	if !reflect.DeepEqual(blob, raw) {

		t.Fatalf("Error Serializing blobs, the serialized and deserialized versions are not the same:\n\n %v \n\n %v \n\n %v", blob, serializedblob, raw)
	}

}

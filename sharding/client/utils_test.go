package client

import (
	"math/rand"
	"reflect"
	"testing"
)

var testbody interface{}

func buildtxblobs(size int64) []interface{} {
	tempbody := make([]interface{}, size)
	for i := int64(0); i < size; i++ {
		tempbody[i] = buildblob(size)

	}
	return tempbody
}

func buildblob(size int64) []interface{} {

	tempbody := make([]interface{}, size)
	for i := int64(0); i < size; i++ {
		tempbody[i] = byte(rand.Int())

	}

	return tempbody

}
func TestConvertInterface(t *testing.T) {
	var slice interface{}
	slice = []interface{}{0, 1, 2, 3, 4, 5}
	convertedValue, err := convertInterface(slice, reflect.Slice)
	if err != nil {
		t.Fatalf("Error: %v %v", err, convertedValue)
	}

}
func TestSize(t *testing.T) {
	size := int64(84)
	blob := buildtxblobs(size)
	chunksafterSerialize := size * size / chunkDataSize
	terminalchunk := size * size % chunkDataSize
	if terminalchunk != 0 {
		chunksafterSerialize = chunksafterSerialize + 1
	}
	sizeafterSerialize := chunksafterSerialize * chunkSize
	serializedblob, err := Serialize(blob)
	if err != nil {
		t.Fatalf("Error Serializing blob:%v %v", err, serializedblob)
	}

	if int64(len(serializedblob)) != sizeafterSerialize {

		t.Fatalf("Error Serializing blob,Lengths are not the same: %v , %v", int64(len(serializedblob)), sizeafterSerialize)

	}

}
func TestSerializeAndDeserializeblob(t *testing.T) {

	blob := buildtxblobs(31)

	serializedblob, err := Serialize(blob)

	if err != nil {
		t.Fatalf("Error Serializing blob:%v %v", err, serializedblob)
	}
	err2 := Deserializebody(serializedblob, &testbody)
	if err2 != nil {
		t.Fatalf("Error Serializing blob:%v", err2)
	}

	if !reflect.DeepEqual(blob, testbody) {

		t.Fatalf("Error Serializing blob with %v %v", blob, testbody)
	}

}

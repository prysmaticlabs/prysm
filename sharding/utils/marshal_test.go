package utils

import (
	"math/rand"
	"reflect"
	"testing"
)

func buildRawBlob(size int64) []RawBlob {
	tempbody := make([]RawBlob, size)
	for i := int64(0); i < size; i++ {
		var rawblob RawBlob
		rawblob.data = buildBlob(size)
		flagset := byte(rand.Int()) >> 7
		if flagset == byte(1) {
			rawblob.flags.skipEvmExecution = true

		}
		tempbody[i] = rawblob
	}

	return tempbody
}

func buildBlob(size int64) []byte {
	tempbody := make([]byte, size)
	for i := int64(0); i < size; i++ {
		tempbody[i] = byte(rand.Int())
	}

	return tempbody
}

func TestSerializeBlob(t *testing.T) {
	for i := 1; i < 300; i++ {
		blobSize := int64(i)
		var rawBlob RawBlob
		rawBlob.data = buildBlob(blobSize)

		chunksAfterSerialize := blobSize / chunkDataSize
		terminalChunk := blobSize % chunkDataSize
		if terminalChunk != 0 {
			chunksAfterSerialize = chunksAfterSerialize + 1
		}

		serializedBlobSize := chunksAfterSerialize * chunkSize
		serializedBlob, err := SerializeBlob(rawBlob)

		if err != nil {
			t.Errorf("Error serializing blob: %v\n %v", err, serializedBlob)
		}

		if int64(len(serializedBlob)) != serializedBlobSize {
			t.Errorf("Size of serialized blob is %v but should be %v", len(serializedBlob), serializedBlobSize)
		}
	}
}

func TestSize(t *testing.T) {
	for i := 0; i < 300; i++ {
		size := int64(i)
		blob := buildRawBlob(size)
		chunksafterSerialize := size / chunkDataSize
		terminalchunk := size % chunkDataSize
		if terminalchunk != 0 {
			chunksafterSerialize = chunksafterSerialize + 1
		}
		chunksafterSerialize = chunksafterSerialize * size
		sizeafterSerialize := chunksafterSerialize * chunkSize

		drefbody := make([]*RawBlob, len(blob))
		for s := 0; s < len(blob); s++ {
			drefbody[s] = &(blob[s])
		}
		serializedblob, err := Serialize(drefbody)
		if err != nil {
			t.Errorf("Error Serializing blob:%v\n %v", err, serializedblob)
		}

		if int64(len(serializedblob)) != sizeafterSerialize {
			t.Errorf("Error Serializing blobs the lengths are not the same:\n %d \n %d", int64(len(serializedblob)), sizeafterSerialize)
		}
	}
}

func TestSerializeAndDeserializeblob(t *testing.T) {

	for i := 1; i < 300; i++ {

		blob := buildRawBlob(int64(i))

		drefbody := make([]*RawBlob, len(blob))
		for s := 0; s < len(blob); s++ {
			drefbody[s] = &(blob[s])
		}

		serializedblob, err := Serialize(drefbody)

		if err != nil {
			t.Errorf("Error Serializing blob at index %d:\n%v\n%v", i, err, serializedblob)
		}
		raw, err2 := Deserialize(serializedblob)
		if err2 != nil {
			t.Errorf("Error Serializing blob at index %d:\n%v due to \n%v", i, raw, err2)
		}

		if !reflect.DeepEqual(blob, raw) {
			t.Errorf("Error Serializing blobs at index %d, the serialized and deserialized versions are not the same:\n\n %v \n\n %v \n\n %v", i, blob, serializedblob, raw)
		}
	}
}

func TestSkipEvm(t *testing.T) {
	data := make([]byte, 64)

	// Set the indicator byte of the second chunk so that the first flag bit (SKIP_EVM) is true and the length bits equal 1
	data[32] = 0x81
	rawBlobs, err := Deserialize(data)
	if err != nil {
		t.Errorf("Deserialize failed: %v", err)
	}

	if len(rawBlobs) != 1 {
		t.Errorf("Length of blobs incorrect: %v", err)
	}

	if !rawBlobs[0].flags.skipEvmExecution {
		t.Errorf("SKIP_EVM flag is not true")
	}

	if len(rawBlobs[0].data) != 32 {
		t.Errorf("blob size is not 32: %v", len(rawBlobs[0].data))
	}
}

func TestNotSkipEvm(t *testing.T) {
	// create 64 byte array with the isSkipEVM flag turned on
	data := make([]byte, 64)

	// Set the indicator byte of the second chunk so that no flag is true and the length bits equal 2
	data[32] = 0x02
	rawBlobs, err := Deserialize(data)
	if err != nil {
		t.Errorf("Deserialize failed: %v", err)
	}

	if len(rawBlobs) != 1 {
		t.Errorf("Length of blobs incorrect: %v", err)
	}

	if rawBlobs[0].flags.skipEvmExecution {
		t.Errorf("SKIP_EVM flag is true")
	}

	if len(rawBlobs[0].data) != 33 {
		t.Errorf("blob size is not 33: %v", len(rawBlobs[0].data))
	}
}


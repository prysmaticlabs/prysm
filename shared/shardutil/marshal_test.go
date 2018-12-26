package shardutil

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

func TestDeserializeSkipEvm(t *testing.T) {
	data := make([]byte, 64)

	// Set the indicator byte of the second chunk so that the first flag bit (SKIP_EVM) is true and the length bits equal 1
	data[32] = 0x81
	rawBlobs, err := Deserialize(data)
	if err != nil {
		t.Errorf("Deserialize failed: %v", err)
	}

	if len(rawBlobs) != 1 {
		t.Errorf("Length of blobs incorrect: %d", len(rawBlobs))
	}

	if !rawBlobs[0].flags.skipEvmExecution {
		t.Errorf("SKIP_EVM flag is not true")
	}

	blobSize := 32
	if len(rawBlobs[0].data) != blobSize {
		t.Errorf("blob size should be %d but is %d", blobSize, len(rawBlobs[0].data))
	}
}

func TestDeserializeSkipEvmFalse(t *testing.T) {
	// create 64 byte array with the isSkipEVM flag turned on
	data := make([]byte, 64)

	// Set the indicator byte of the second chunk so that no flag is true and the length bits equal 2
	data[32] = 0x02
	rawBlobs, err := Deserialize(data)
	if err != nil {
		t.Errorf("Deserialize failed: %v", err)
	}

	if len(rawBlobs) != 1 {
		t.Errorf("Length of blobs incorrect: %d", len(rawBlobs))
	}

	if rawBlobs[0].flags.skipEvmExecution {
		t.Errorf("SKIP_EVM flag is true")
	}

	blobSize := 33
	if len(rawBlobs[0].data) != blobSize {
		t.Errorf("blob size should be %d but is %d", blobSize, len(rawBlobs[0].data))
	}
}

func TestSerializeSkipEvm(t *testing.T) {
	rawBlobs := make([]*RawBlob, 1)
	rawBlobs[0] = &RawBlob{data: make([]byte, 32)}
	rawBlobs[0].data[31] = byte(1)
	rawBlobs[0].flags.skipEvmExecution = true

	data, err := Serialize(rawBlobs)
	if err != nil {
		t.Errorf("Serialize failed: %v", err)
	}

	dataSize := 64
	if len(data) != dataSize {
		t.Errorf("Length of serialized data incorrect. Should be %d but is %d", dataSize, len(data))
	}

	if data[0] != 0 {
		t.Errorf("Indicating byte for first chunk should be %#x but is %#x", 0, data[0])
	}

	indicatingByte := byte(0x81)
	if data[32] != indicatingByte {
		t.Errorf("Indicating byte for second chunk should be %#x but is %#x", indicatingByte, data[32])
	}
}

func TestSerializeSkipEvmFalse(t *testing.T) {
	rawBlobs := make([]*RawBlob, 1)
	rawBlobs[0] = &RawBlob{data: make([]byte, 31)}

	data, err := Serialize(rawBlobs)
	if err != nil {
		t.Errorf("Serialize failed: %v", err)
	}

	blobSize := 32
	if len(data) != blobSize {
		t.Errorf("Length of serialized data incorrect. Should be %d but is %d", blobSize, len(data))
	}

	indicatingByte := byte(0x1f)
	if data[0] != indicatingByte {
		t.Errorf("Indicating byte for first chunk should be %#x but is %#x", indicatingByte, data[0])
	}
}

func TestSerializeTestData(t *testing.T) {
	rawBlobs := make([]*RawBlob, 1)
	rawBlobs[0] = &RawBlob{data: make([]byte, 60)}
	blobData := rawBlobs[0].data
	for i := 0; i < len(blobData); i++ {
		blobData[i] = byte(i)
	}

	data, err := Serialize(rawBlobs)
	if err != nil {
		t.Errorf("Serialize failed: %v", err)
	}

	blobSize := 64
	if len(data) != blobSize {
		t.Errorf("Length of serialized data incorrect. Should be %d but is %d", blobSize, len(data))
	}

	indicatingByte := byte(0x1D)
	if data[32] != indicatingByte {
		t.Errorf("Indicating byte for second chunk should be %#x but is %#x", indicatingByte, data[32])
	}

	for i := 1; i < 32; i++ {
		if data[i] != byte(i-1) {
			t.Errorf("Data byte incorrect. Should be %#x but is %#x", byte(i-1), data[i])
		}
	}

	for i := 33; i < 62; i++ {
		if data[i] != byte(i-2) {
			t.Errorf("Data byte incorrect. Should be %#x but is %#x", byte(i-2), data[i])
		}
	}
}

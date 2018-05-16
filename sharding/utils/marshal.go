package utils

import (
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	chunkSize     = int64(32)
	indicatorSize = int64(1)
	chunkDataSize = chunkSize - indicatorSize
)

// Flags to add to chunk delimiter.
type Flags struct {
	skipEvmExecution bool
}

// RawBlob type which will contain flags and data for serialization.
type RawBlob struct {
	flags Flags
	data  []byte
}

// NewRawBlob builds a raw blob from any interface by using RLP encoding
func NewRawBlob(i interface{}, skipEvm bool) (*RawBlob, error) {
	data, err := rlp.EncodeToBytes(i)
	if err != nil {
		return nil, fmt.Errorf("RLP encoding was a failure:%v", err)
	}
	return &RawBlob{data: data, flags: Flags{skipEvmExecution: skipEvm}}, nil
}

// ConvertFromRawBlob converts raw blob back from a byte array to its interface
func ConvertFromRawBlob(blob *RawBlob, i interface{}) error {
	data := (*blob).data
	err := rlp.DecodeBytes(data, i)
	if err != nil {
		return fmt.Errorf("RLP decoding was a failure:%v", err)
	}

	return nil
}

// SerializeBlob parses the blob and serializes it appropriately.
func SerializeBlob(cb RawBlob) ([]byte, error) {

	length := int64(len(cb.data))
	terminalLength := length % chunkDataSize
	chunksNumber := length / chunkDataSize
	indicatorByte := make([]byte, 1)
	indicatorByte[0] = 0
	if cb.flags.skipEvmExecution {
		indicatorByte[0] |= (1 << 7)
	}
	tempBody := []byte{}

	// if blob is less than 31 bytes, it adds the indicator chunk and pads the remaining empty bytes to the right

	if chunksNumber == 0 {
		paddedBytes := make([]byte, (chunkDataSize - length))
		indicatorByte[0] = byte(terminalLength)
		if cb.flags.skipEvmExecution {
			indicatorByte[0] |= (1 << 7)
		}
		tempBody = append(indicatorByte, append(cb.data, paddedBytes...)...)
		return tempBody, nil
	}

	//if there is no need to pad empty bytes, then the indicator byte is added as 00011111
	// Then this chunk is returned to the main Serialize function

	if terminalLength == 0 {

		for i := int64(1); i < chunksNumber; i++ {
			// This loop loops through all non-terminal chunks and add a indicator byte of 00000000, each chunk
			// is created by appending the indcator byte to the data chunks. The data chunks are separated into sets of
			// 31

			tempBody = append(tempBody,
				append(indicatorByte,
					cb.data[(i-1)*chunkDataSize:i*chunkDataSize]...)...)

		}
		indicatorByte[0] = byte(chunkDataSize)
		if cb.flags.skipEvmExecution {
			indicatorByte[0] |= (1 << 7)
		}

		// Terminal chunk has its indicator byte added, chunkDataSize*chunksNumber refers to the total size of the blob
		tempBody = append(tempBody,
			append(indicatorByte,
				cb.data[(chunksNumber-1)*chunkDataSize:chunkDataSize*chunksNumber]...)...)

		return tempBody, nil

	}

	// This loop loops through all non-terminal chunks and add a indicator byte of 00000000, each chunk
	// is created by appending the indcator byte to the data chunks. The data chunks are separated into sets of
	// 31

	for i := int64(1); i <= chunksNumber; i++ {

		tempBody = append(tempBody,
			append(indicatorByte,
				cb.data[(i-1)*chunkDataSize:i*chunkDataSize]...)...)

	}
	// Appends indicator bytes to terminal-chunks , and if the index of the chunk delimiter is non-zero adds it to the chunk.
	// Also pads empty bytes to the terminal chunk.chunkDataSize*chunksNumber refers to the total size of the blob.
	// finalchunkIndex refers to the index of the last data byte
	indicatorByte[0] = byte(terminalLength)
	if cb.flags.skipEvmExecution {
		indicatorByte[0] |= (1 << 7)
	}
	tempBody = append(tempBody,
		append(indicatorByte,
			cb.data[chunkDataSize*chunksNumber:length]...)...)

	emptyBytes := make([]byte, (chunkDataSize - terminalLength))
	tempBody = append(tempBody, emptyBytes...)

	return tempBody, nil

}

// Serialize takes a set of blobs and converts them to a single byte array.
func Serialize(rawblobs []*RawBlob) ([]byte, error) {
	length := int64(len(rawblobs))

	serialisedData := []byte{}

	//Loops through all the blobs and serializes them into chunks
	for i := int64(0); i < length; i++ {

		data := *rawblobs[i]
		refinedData, err := SerializeBlob(data)
		if err != nil {
			return nil, fmt.Errorf("Index %v :  %v", i, err)
		}
		serialisedData = append(serialisedData, refinedData...)

	}
	return serialisedData, nil
}

// Deserialize results in the byte array being deserialised and separated into its respective interfaces.
func Deserialize(data []byte) ([]RawBlob, error) {

	length := int64(len(data))
	chunksNumber := length / chunkSize
	indicatorByte := byte(0)
	tempBody := RawBlob{}
	var deserializedBlob []RawBlob

	// This separates the byte array into its separate blobs
	for i := int64(1); i <= chunksNumber; i++ {
		indicatorIndex := (i - 1) * chunkSize

		// Tests if the chunk delimiter is zero, if it is it will append the data chunk
		// to tempBody
		if data[indicatorIndex] == indicatorByte || data[indicatorIndex] == byte(128) {
			tempBody.data = append(tempBody.data, data[(indicatorIndex+1):(i)*chunkSize]...)

		} else if data[indicatorIndex] == byte(31) || data[indicatorIndex] == byte(159) {
			if data[indicatorIndex] == byte(159) {
				tempBody.flags.skipEvmExecution = true
			}
			tempBody.data = append(tempBody.data, data[(indicatorIndex+1):indicatorIndex+1+chunkDataSize]...)
			deserializedBlob = append(deserializedBlob, tempBody)
			tempBody = RawBlob{}

		} else {
			// Since the chunk delimiter in non-zero now we can infer that it is a terminal chunk and
			// add it and append to the deserializedblob slice. The tempBody signifies a single deserialized blob
			terminalIndex := int64(data[indicatorIndex])
			//Check if EVM flag is equal to 1
			flagindex := data[indicatorIndex] >> 7
			if flagindex == byte(1) {
				terminalIndex = int64(data[indicatorIndex]) - 128
				tempBody.flags.skipEvmExecution = true
			}
			tempBody.data = append(tempBody.data, data[(indicatorIndex+1):(indicatorIndex+1+terminalIndex)]...)
			deserializedBlob = append(deserializedBlob, tempBody)
			tempBody = RawBlob{}

		}

	}

	return deserializedBlob, nil

}

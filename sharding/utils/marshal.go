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

// NewRawBlob builds a raw blob from any interface by using
// RLP encoding.
func NewRawBlob(i interface{}, skipEvm bool) (*RawBlob, error) {
	data, err := rlp.EncodeToBytes(i)
	if err != nil {
		return nil, fmt.Errorf("RLP encoding was a failure:%v", err)
	}
	return &RawBlob{data: data, flags: Flags{skipEvmExecution: skipEvm}}, nil
}

// ConvertFromRawBlob converts raw blob back from a byte array
// to its interface.
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

	// if blob is less than 31 bytes, adds the indicator chunk
	// and pad the remaining empty bytes to the right.
	if chunksNumber == 0 {
		paddedBytes := make([]byte, (chunkDataSize - length))
		indicatorByte[0] = byte(terminalLength)
		if cb.flags.skipEvmExecution {
			indicatorByte[0] |= (1 << 7)
		}
		return append(indicatorByte, append(cb.data, paddedBytes...)...), nil
	}

	// if there is no need to pad empty bytes, then the indicator byte
	// is added as 0001111 and this chunk is returned to the
	// main Serialize function.
	if terminalLength == 0 {

		for i := int64(1); i < chunksNumber; i++ {
			// This loop loops through all non-terminal chunks and add a indicator
			// byte of 00000000, each chunk is created by appending the indicator
			// byte to the data chunks. The data chunks are separated into sets of
			// 31 bytes.

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

	// This loop loops through all non-terminal chunks and add a indicator byte
	// of 00000000, each chunk is created by appending the indcator byte
	// to the data chunks. The data chunks are separated into sets of 31.
	for i := int64(1); i <= chunksNumber; i++ {
		tempBody = append(tempBody,
			append(indicatorByte,
				cb.data[(i-1)*chunkDataSize:i*chunkDataSize]...)...)
	}

	// Appends indicator bytes to terminal-chunks, and if the index of the chunk
	// delimiter is non-zero adds it to the chunk. Also pads empty bytes to
	// the terminal chunk. chunkDataSize*chunksNumber refers to the total
	// size of the blob. finalchunkIndex refers to the index of the last data byte.
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
	numBlobs := int64(len(rawblobs))
	dataSize := 0

	serializedData := make([][]byte, numBlobs)

	// Loop through all blobs and store the serialized data
	for i := int64(0); i < numBlobs; i++ {
		data := *rawblobs[i]
		refinedData, err := SerializeBlob(data)
		if err != nil {
			return nil, fmt.Errorf("Index %v: %v", i, err)
		}

		serializedData[i] = refinedData
		dataSize += len(refinedData)
	}

	returnData := make([]byte, 0, dataSize)
	for i := int64(0); i < numBlobs; i++ {
		returnData = append(returnData, serializedData[i]...)
	}

	return returnData, nil
}

func isSkipEvm(indicator byte) bool {
	return indicator & 0xE0 >> 7 == 1
}
func getDatabyteLength(indicator byte) int {
	return int(indicator & 0x1F)
}

type SerializedBlob struct {
	numNonTerminalChunks int
	terminalLength       int
}

// Deserialize results in the byte array being deserialised and
// separated into its respective interfaces.
func Deserialize(data []byte) ([]RawBlob, error) {
	chunksNumber := len(data) / int(chunkSize)
	serializedBlobs := []SerializedBlob{}
	numPartitions := 0

	// first iterate through every chunk and identify blobs and their length
	for i := 0; i < chunksNumber; i++ {
		indicatorIndex := i * int(chunkSize)
		databyteLength := getDatabyteLength(data[indicatorIndex])

		// if indicator is non-terminal, increase blobSize by 31
		if databyteLength == 0 {
			numPartitions += 1
		} else {
			// if indicator is terminal, increase blobSize by that number and reset
			serializedBlob := SerializedBlob{
				numNonTerminalChunks: numPartitions,
				terminalLength:       databyteLength,
			}
			serializedBlobs = append(serializedBlobs, serializedBlob)
			numPartitions = 0
		}
	}

	// for each block, construct the data byte array
	deserializedBlob := make([]RawBlob, 0, len(serializedBlobs))
	currentByte := 0
	for i := 0; i < len(serializedBlobs); i++ {
		numNonTerminalChunks := serializedBlobs[i].numNonTerminalChunks
		terminalLength := serializedBlobs[i].terminalLength

		blob := RawBlob{}
		blob.data = make([]byte, 0, numNonTerminalChunks * 31 + terminalLength)

		// append data from non-terminal chunks
		for chunk := 0; chunk < numNonTerminalChunks; chunk++ {
			dataBytes := data[currentByte+1:currentByte+32]
			blob.data = append(blob.data, dataBytes...)
			currentByte += 32
		}

		if isSkipEvm(data[currentByte]) {
			blob.flags.skipEvmExecution = true
		}

		// append data from terminal chunk
		dataBytes := data[currentByte+1:currentByte+terminalLength+1]
		blob.data = append(blob.data, dataBytes...)
		currentByte += 32

		deserializedBlob = append(deserializedBlob, blob)
	}

	return deserializedBlob, nil
}

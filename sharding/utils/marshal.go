package utils

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	chunkSize     = int64(32)
	indicatorSize = int64(1)
	chunkDataSize = chunkSize - indicatorSize
)

type Flags struct {
	skipEvmExecution bool
}

type RawBlob struct {
	flags Flags
	data  []byte
}

// ConvertInterface converts inputted interface to the required type of interface, ex: slice.
func ConvertInterfacetoBytes(arg interface{}) ([]byte, error) {
	val := reflect.ValueOf(arg)
	if val.Kind() == reflect.Slice {

		length := val.Len()
		newtype := make([]byte, length)
		for i := 0; i < length; i++ {
			newtype[i] = val.Index(i).Interface().(byte)
		}

		return newtype, nil

	}
	err := errors.New("Interface Conversion a failure")
	return nil, err

}

func convertbyteToInterface(arg []byte) []interface{} {
	length := int64(len(arg))
	newtype := make([]interface{}, length)
	for i, v := range arg {
		newtype[i] = v
	}

	return newtype
}

func ConvertToInterface(arg interface{}) []interface{} {
	val := reflect.ValueOf(arg)
	length := val.Len()
	newtype := make([]interface{}, length)
	for i := 0; i < length; i++ {
		newtype[i] = val.Index(i)
	}

	return newtype
}

// serializeBlob parses the blob and serializes it appropriately.
func serializeBlob(cb RawBlob) ([]byte, error) {

	length := int64(len(cb.data))
	terminalLength := length % chunkDataSize
	chunksNumber := length / chunkDataSize
	indicatorByte := make([]byte, 1)
	indicatorByte[0] = 0
	if cb.flags.skipEvmExecution {
		indicatorByte[0] |= (1 << 7)
	}
	tempbody := []byte{}

	// if blob is less than 31 bytes, it adds the indicator chunk and pads the remaining empty bytes to the right

	if chunksNumber == 0 {
		paddedbytes := make([]byte, (chunkDataSize - length))
		indicatorByte[0] = byte(terminalLength)
		if cb.flags.skipEvmExecution {
			indicatorByte[0] |= (1 << 7)
		}
		tempbody = append(indicatorByte, append(cb.data, paddedbytes...)...)
		return tempbody, nil
	}

	//if there is no need to pad empty bytes, then the indicator byte is added as 00011111
	// Then this chunk is returned to the main Serialize function

	if terminalLength == 0 {

		for i := int64(1); i < chunksNumber; i++ {
			// This loop loops through all non-terminal chunks and add a indicator byte of 00000000, each chunk
			// is created by appending the indcator byte to the data chunks. The data chunks are separated into sets of
			// 31

			tempbody = append(tempbody,
				append(indicatorByte,
					cb.data[(i-1)*chunkDataSize:i*chunkDataSize]...)...)

		}
		indicatorByte[0] = byte(chunkDataSize)
		if cb.flags.skipEvmExecution {
			indicatorByte[0] |= (1 << 7)
		}

		// Terminal chunk has its indicator byte added, chunkDataSize*chunksNumber refers to the total size of the blob
		tempbody = append(tempbody,
			append(indicatorByte,
				cb.data[(chunksNumber-1)*chunkDataSize:chunkDataSize*chunksNumber]...)...)

		return tempbody, nil

	}

	// This loop loops through all non-terminal chunks and add a indicator byte of 00000000, each chunk
	// is created by appending the indcator byte to the data chunks. The data chunks are separated into sets of
	// 31

	for i := int64(1); i <= chunksNumber; i++ {

		tempbody = append(tempbody,
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
	tempbody = append(tempbody,
		append(indicatorByte,
			cb.data[chunkDataSize*chunksNumber:length]...)...)

	emptyBytes := make([]byte, (chunkDataSize - terminalLength))
	tempbody = append(tempbody, emptyBytes...)

	return tempbody, nil

}

// Serialize takes a set of blobs and converts them to a single byte array.
func Serialize(rawblobs []RawBlob) ([]byte, error) {
	length := int64(len(rawblobs))

	if length == 0 {
		return nil, fmt.Errorf("Validation failed: Collation Body has to be a non-zero value")
	}
	serialisedData := []byte{}

	//Loops through all the blobs and serializes them into chunks
	for i := int64(0); i < length; i++ {

		data := rawblobs[i]
		refinedData, err := serializeBlob(data)
		if err != nil {
			return nil, fmt.Errorf("Index %v :  %v", i, err)
		}
		serialisedData = append(serialisedData, refinedData...)

	}
	return serialisedData, nil
}

// Deserialize results in the byte array being deserialised and separated into its respective interfaces.
func Deserialize(collationbody []byte) ([]RawBlob, error) {

	length := int64(len(collationbody))
	chunksNumber := length / chunkSize
	indicatorByte := byte(0)
	tempbody := RawBlob{}
	var deserializedblob []RawBlob

	// This separates the byte array into its separate blobs
	for i := int64(1); i <= chunksNumber; i++ {
		indicatorIndex := (i - 1) * chunkSize

		// Tests if the chunk delimiter is zero, if it is it will append the data chunk
		// to tempbody
		if collationbody[indicatorIndex] == indicatorByte || collationbody[indicatorIndex] == byte(128) {
			tempbody.data = append(tempbody.data, collationbody[(indicatorIndex+1):(i)*chunkSize]...)

		} else if collationbody[indicatorIndex] == byte(31) || collationbody[indicatorIndex] == byte(159) {
			if collationbody[indicatorIndex] == byte(159) {
				tempbody.flags.skipEvmExecution = true
			}
			tempbody.data = append(tempbody.data, collationbody[(indicatorIndex+1)])
			deserializedblob = append(deserializedblob, tempbody)
			tempbody = RawBlob{}

		} else {
			// Since the chunk delimiter in non-zero now we can infer that it is a terminal chunk and
			// add it and append to the deserializedblob slice. The tempbody signifies a single deserialized blob
			terminalIndex := int64(collationbody[indicatorIndex])
			//Check if EVM flag is equal to 1
			flagindex := collationbody[indicatorIndex] >> 7
			if flagindex == byte(1) {
				terminalIndex = int64(collationbody[indicatorIndex]) - 128
				tempbody.flags.skipEvmExecution = true
			}
			tempbody.data = append(tempbody.data, collationbody[(indicatorIndex+1):(indicatorIndex+1+terminalIndex)]...)
			deserializedblob = append(deserializedblob, tempbody)
			tempbody = RawBlob{}

		}

	}

	return deserializedblob, nil

}

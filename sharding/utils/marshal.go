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

// ConvertInterface converts inputted interface to the required type of interface, ex: slice.
func ConvertInterface(arg interface{}, kind reflect.Kind) ([]interface{}, error) {
	val := reflect.ValueOf(arg)
	if val.Kind() == kind {

		return val.Interface().([]interface{}), nil

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

func interfacetoByte(arg []interface{}) []byte {
	length := int64(len(arg))
	newtype := make([]byte, length)
	for i, v := range arg {
		newtype[i] = v.(byte)
	}

	return newtype
}

// serializeBlob parses the blob and serializes it appropriately.
func serializeBlob(cb interface{}) ([]byte, error) {

	interfaceblob, err := ConvertInterface(cb, reflect.Slice)
	if err != nil {
		return nil, fmt.Errorf("Error: %v", err)
	}
	blob := interfacetoByte(interfaceblob)
	length := int64(len(blob))
	terminalLength := length % chunkDataSize
	chunksNumber := length / chunkDataSize
	indicatorByte := make([]byte, 1)
	indicatorByte[0] = 0
	tempbody := []byte{}

	// if blob is less than 31 bytes, it adds the indicator chunk and pads the remaining empty bytes to the right

	if chunksNumber == 0 {
		paddedbytes := make([]byte, (chunkDataSize - length))
		indicatorByte[0] = byte(terminalLength)
		tempbody = append(indicatorByte, append(blob, paddedbytes...)...)
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
					blob[(i-1)*chunkDataSize:i*chunkDataSize]...)...)

		}
		indicatorByte[0] = byte(chunkDataSize)

		// Terminal chunk has its indicator byte added, chunkDataSize*chunksNumber refers to the total size of the blob
		tempbody = append(tempbody,
			append(indicatorByte,
				blob[(chunksNumber-1)*chunkDataSize:chunkDataSize*chunksNumber]...)...)

		return tempbody, nil

	}

	// This loop loops through all non-terminal chunks and add a indicator byte of 00000000, each chunk
	// is created by appending the indcator byte to the data chunks. The data chunks are separated into sets of
	// 31

	for i := int64(1); i <= chunksNumber; i++ {

		tempbody = append(tempbody,
			append(indicatorByte,
				blob[(i-1)*chunkDataSize:i*chunkDataSize]...)...)

	}
	// Appends indicator bytes to terminal-chunks , and if the index of the chunk delimiter is non-zero adds it to the chunk.
	// Also pads empty bytes to the terminal chunk.chunkDataSize*chunksNumber refers to the total size of the blob.
	// finalchunkIndex refers to the index of the last data byte
	indicatorByte[0] = byte(terminalLength)
	tempbody = append(tempbody,
		append(indicatorByte,
			blob[chunkDataSize*chunksNumber:length]...)...)

	emptyBytes := make([]byte, (chunkDataSize - terminalLength))
	tempbody = append(tempbody, emptyBytes...)

	return tempbody, nil

}

// Serialize takes a set of blobs and converts them to a single byte array.
func Serialize(rawtx []interface{}) ([]byte, error) {
	length := int64(len(rawtx))

	if length == 0 {
		return nil, fmt.Errorf("Validation failed: Collation Body has to be a non-zero value")
	}
	serialisedData := []byte{}

	//Loops through all the blobs and serializes them into chunks
	for i := int64(0); i < length; i++ {

		data := rawtx[i]
		refinedData, err := serializeBlob(data)
		if err != nil {
			return nil, fmt.Errorf("Error: %v at index: %v", i, err)
		}
		serialisedData = append(serialisedData, refinedData...)

	}
	return serialisedData, nil
}

// Deserialize results in the byte array being deserialised and separated into its respective interfaces.
func Deserialize(collationbody []byte, rawtx interface{}) error {

	length := int64(len(collationbody))
	chunksNumber := length / chunkSize
	indicatorByte := byte(0)
	tempbody := []byte{}
	var deserializedblob []interface{}

	// This separates the byte array into its separate blobs
	for i := int64(1); i <= chunksNumber; i++ {
		indicatorIndex := (i - 1) * chunkSize
		// Tests if the chunk delimiter is zero, if it is it will append the data chunk
		// to tempbody
		if collationbody[indicatorIndex] == indicatorByte {
			tempbody = append(tempbody, collationbody[(indicatorIndex+1):(i)*chunkSize]...)

		} else if collationbody[indicatorIndex] == byte(1) {

			tempbody = append(tempbody, collationbody[(indicatorIndex+1)])
			deserializedblob = append(deserializedblob, convertbyteToInterface(tempbody))
			tempbody = []byte{}

		} else {
			// Since the chunk delimiter in non-zero now we can infer that it is a terminal chunk and
			// add it and append to the deserializedblob slice. The tempbody signifies a single deserialized blob
			terminalIndex := int64(collationbody[indicatorIndex])
			tempbody = append(tempbody, collationbody[(indicatorIndex+1):(indicatorIndex+1+terminalIndex)]...)
			deserializedblob = append(deserializedblob, convertbyteToInterface(tempbody))
			tempbody = []byte{}

		}

	}

	*rawtx.(*interface{}) = deserializedblob

	return nil

}

package client

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	//"runtime"

	"github.com/ethereum/go-ethereum/log"
)

var (
	collationsizelimit = int64(math.Pow(float64(2), float64(20)))
	chunkSize          = int64(32)
	indicatorSize      = int64(1)
	numberOfChunks     = collationsizelimit / chunkSize
	chunkDataSize      = chunkSize - indicatorSize
)

// convertInterface converts inputted interface to the required type, ex: slice.
func convertInterface(arg interface{}, kind reflect.Kind) (reflect.Value, error) {
	val := reflect.ValueOf(arg)
	if val.Kind() == kind {
		return val, nil

	}
	err := errors.New("Interface Conversion a failure")
	return val, err
}

func convertbyteToInterface(arg []byte) []interface{} {
	length := int64(len(arg))
	newtype := make([]interface{}, length)
	for i, v := range arg {
		newtype[i] = v
	}

	return newtype
}

// serializeBlob parses the blob and serializes it appropriately.
func serializeBlob(cb interface{}) ([]byte, error) {

	blob, err := convertInterface(cb, reflect.Slice)
	if err != nil {
		return nil, fmt.Errorf("Error: %v", err)
	}
	length := int64(blob.Len())
	terminalLength := length % chunkDataSize
	chunksNumber := length / chunkDataSize
	finalchunkIndex := length - 1
	indicatorByte := make([]byte, 1)
	indicatorByte[0] = 0
	tempbody := []byte{}

	// if blob is less than 31 bytes, it adds the indicator chunk and pads the remaining empty bytes to the right

	if chunksNumber == 0 {
		paddedbytes := make([]byte, (length - terminalLength))
		indicatorByte[0] = byte(terminalLength)
		tempbody = append(indicatorByte, append(blob.Bytes(), paddedbytes...)...)
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
					blob.Bytes()[(i-1)*chunkDataSize:i*chunkDataSize]...)...)

		}
		indicatorByte[0] = byte(chunkDataSize)

		// Terminal chunk has its indicator byte added, chunkDataSize*chunksNumber refers to the total size of the blob
		tempbody = append(tempbody,
			append(indicatorByte,
				blob.Bytes()[(chunksNumber-1)*chunkDataSize:chunkDataSize*chunksNumber]...)...)

		return tempbody, nil

	}

	// This loop loops through all non-terminal chunks and add a indicator byte of 00000000, each chunk
	// is created by appending the indcator byte to the data chunks. The data chunks are separated into sets of
	// 31

	for i := int64(1); i <= chunksNumber; i++ {

		tempbody = append(tempbody,
			append(indicatorByte,
				blob.Bytes()[(i-1)*chunkDataSize:i*chunkDataSize]...)...)

	}
	// Appends indicator bytes to terminal-chunks , and if the index of the chunk delimiter is non-zero adds it to the chunk.
	// Also pads empty bytes to the terminal chunk.chunkDataSize*chunksNumber refers to the total size of the blob.
	// finalchunkIndex refers to the index of the last data byte
	indicatorByte[0] = byte(terminalLength)
	tempbody = append(tempbody,
		append(indicatorByte,
			blob.Bytes()[chunkDataSize*chunksNumber-1:finalchunkIndex]...)...)

	emptyBytes := make([]byte, (chunkDataSize - terminalLength))
	tempbody = append(tempbody, emptyBytes...)

	return tempbody, nil

}

// Serialize takes a set of transaction blobs and converts them to a single byte array.
func Serialize(rawtx []interface{}) ([]byte, error) {
	length := int64(len(rawtx))

	if length == 0 {
		return nil, fmt.Errorf("Validation failed: Collation Body has to be a non-zero value")
	}
	serialisedData := []byte{}

	//Loops through all the blobs and serializes them into chunks
	for i := int64(0); i < length; i++ {

		blobLength := int64(len(serialisedData))
		data := rawtx[i]
		refinedData, err := serializeBlob(data)
		if err != nil {
			return nil, fmt.Errorf("Error: %v at index: %v %v %v", err, i, data, rawtx)
		}
		serialisedData = append(serialisedData, refinedData...)

		if int64(len(serialisedData)) > collationsizelimit {
			log.Info(fmt.Sprintf("The total number of interfaces added to the collation body are: %d", i))
			serialisedData = serialisedData[:blobLength]
			return serialisedData, nil

		}

	}
	return serialisedData, nil
}

// Deserializebody results in the Collation body being deserialised and separated into its respective interfaces.
func Deserializebody(collationbody []byte, rawtx interface{}) error {

	length := int64(len(collationbody))
	chunksNumber := length / chunkSize
	indicatorByte := byte(0)
	tempbody := []byte{}
	deserializedblob := []byte{}

	// This separates the collation body into its respective transaction blobs
	for i := int64(1); i <= chunksNumber; i++ {
		indicatorIndex := (i - 1) * chunkSize
		// Tests if the chunk delimiter is zero, if it is it will append the data chunk
		// to tempbody
		if collationbody[indicatorIndex] == indicatorByte {
			tempbody = append(tempbody, collationbody[(indicatorIndex+1):(i)*chunkSize]...)

			// Since the chunk delimiter in non-zero now we can infer that it is a terminal chunk and
			// add it and append to the rawtx slice. The tempbody signifies a deserialized blob
		} else {
			terminalIndex := int64(collationbody[indicatorIndex])
			tempbody = append(tempbody, collationbody[(indicatorIndex+1):(indicatorIndex+1+terminalIndex)]...)
			deserializedblob = append(deserializedblob, tempbody...)
			tempbody = []byte{}

		}

	}

	*rawtx.(*interface{}) = convertbyteToInterface(deserializedblob)

	return nil

}

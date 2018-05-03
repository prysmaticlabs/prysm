package client

import (
	"errors"
	"fmt"
	"math"
	"reflect"

	"github.com/ethereum/go-ethereum/log"
)

var (
	collationsizelimit = int64(math.Pow(float64(2), float64(20)))
	chunkSize          = int64(32)
	indicatorSize      = int64(1)
	numberOfChunks     = collationsizelimit / chunkSize
	chunkDataSize      = chunkSize - indicatorSize
	totalDatasize      = numberOfChunks * chunkDataSize
)

func convertInterface(arg interface{}, kind reflect.Kind) (reflect.Value, error) {
	val := reflect.ValueOf(arg)
	if val.Kind() == kind {
		return val, nil

	}
	err := errors.New("Interface Conversion a failure")
	return val, err
}

// Parse blob and modify it accordingly

func serializeBlob(cb interface{}) ([]byte, error) {

	blob, err := convertInterface(cb, reflect.Slice)
	if err != nil {
		return nil, fmt.Errorf("Error: %v", err)
	}
	length := int64(blob.Len())
	terminalLength := length % chunkDataSize
	chunksNumber := length / chunkDataSize
	indicatorByte := make([]byte, 1)
	indicatorByte[0] = 0
	tempbody := []byte{}

	// if blob is less than 31 bytes, it adds the indicator chunk and pads the remaining empty bytes to the right

	if chunksNumber == 0 {
		paddedbytes := make([]byte, length-terminalLength)
		indicatorByte[0] = byte(terminalLength)
		tempbody = append(indicatorByte, append(cb.([]byte), paddedbytes...)...)
		return tempbody, nil
	}

	//if there is no need to pad empty bytes, then the indicator byte is added as 00011111

	if terminalLength == 0 {

		for i := int64(1); i < chunksNumber; i++ {
			tempbody = append(tempbody, append(indicatorByte, blob.Bytes()[(i-1)*chunkDataSize:i*chunkDataSize]...)...)

		}
		indicatorByte[0] = byte(chunkDataSize)
		tempbody = append(tempbody, append(indicatorByte, blob.Bytes()[(chunksNumber-1)*chunkDataSize:chunksNumber*chunkDataSize]...)...)
		return tempbody, nil

	}

	// Appends empty indicator bytes to non terminal-chunks
	for i := int64(1); i <= chunksNumber; i++ {
		tempbody = append(tempbody, append(indicatorByte, blob.Bytes()[(i-1)*chunkDataSize:i*chunkDataSize]...)...)

	}
	// Appends indicator bytes to terminal-chunks , and if the index of the chunk delimiter is non-zero adds it to the chunk
	indicatorByte[0] = byte(terminalLength)
	tempbody = append(tempbody, append(indicatorByte, blob.Bytes()[chunksNumber*chunkDataSize:(chunksNumber*chunkDataSize)+(terminalLength+1)]...)...)
	emptyBytes := make([]byte, (chunkDataSize - terminalLength))
	tempbody = append(tempbody, emptyBytes...)

	return tempbody, nil

}

// Serialize takes a set of transactions and converts them to a single byte array
func Serialize(rawtx []interface{}) ([]byte, error) {
	length := int64(len(rawtx))

	if length == 0 {
		return nil, fmt.Errorf("Validation failed: Collation Body has to be a non-zero value")
	}
	serialisedData := []byte{}

	for i := int64(0); i < length; i++ {

		blobLength := int64(len(serialisedData))
		data := rawtx[i]
		refinedData, err := serializeBlob(data)
		if err != nil {
			return nil, fmt.Errorf("Error: %v", err)
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

// Deserializebody results in the Collation body being deserialised and separated into its respective interfaces
func Deserializebody(collationbody []byte, rawtx []interface{}) error {

	length := int64(len(collationbody))
	chunksNumber := chunkSize / length
	indicatorByte := make([]byte, 1)
	indicatorByte[0] = 0
	var txblobs []interface{}
	tempbody := []byte{}

	for i := int64(1); i <= chunksNumber; i++ {

		if reflect.ValueOf(collationbody[(i-1)*chunkSize]) == reflect.ValueOf(indicatorByte) {
			tempbody = append(tempbody, collationbody[((i-1)*chunkSize+1):(i)*chunkSize]...)

		} else {
			terminalIndex := int64(collationbody[(i-1)*chunkSize])
			tempbody = append(tempbody, collationbody[((i-1)*chunkSize+1):((i-1)*chunkSize+2+terminalIndex)]...)
			txblobs = append(txblobs, tempbody)
			tempbody = []byte{}

		}

	}
	rawtx = txblobs

	return nil

}

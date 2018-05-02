package client

import (
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

type txblob []byte

type blob interface {
	length() int64
	serializeBlob() []byte
}

func (cb txblob) length() int64 {

	return int64(len(cb))
}

// Parse blob and modify it accordingly

func (cb txblob) serializeBlob() []byte {

	terminalLength := cb.length() % chunkDataSize
	chunksNumber := cb.length() / chunkDataSize
	indicatorByte := make([]byte, 1)
	indicatorByte[0] = 0
	tempbody := []byte{}

	// if blob is less than 31 bytes, it adds the indicator chunk and pads the remaining empty bytes to the right

	if chunksNumber == 0 {
		paddedbytes := make([]byte, cb.length()-terminalLength)
		indicatorByte[0] = byte(terminalLength)
		tempbody = append(indicatorByte, append(cb, paddedbytes...)...)
		return tempbody
	}

	//if there is no need to pad empty bytes, then the indicator byte is added as 00011111

	if terminalLength == 0 {

		for i := int64(1); i < chunksNumber; i++ {
			tempbody = append(tempbody, append(indicatorByte, cb[(i-1)*chunkDataSize:i*chunkDataSize]...)...)

		}
		indicatorByte[0] = byte(chunkDataSize)
		tempbody = append(tempbody, append(indicatorByte, cb[(chunksNumber-1)*chunkDataSize:chunksNumber*chunkDataSize]...)...)
		return tempbody

	}

	// Appends empty indicator bytes to non terminal-chunks
	for i := int64(1); i <= chunksNumber; i++ {
		tempbody = append(tempbody, append(indicatorByte, cb[(i-1)*chunkDataSize:i*chunkDataSize]...)...)

	}
	// Appends indicator bytes to terminal-chunks , and if the index of the chunk delimiter is non-zero adds it to the chunk
	indicatorByte[0] = byte(terminalLength)
	tempbody = append(tempbody, append(indicatorByte, cb[chunksNumber*chunkDataSize:(chunksNumber*chunkDataSize)+(terminalLength+1)]...)...)
	emptyBytes := make([]byte, (chunkDataSize - cb.length()))
	cb = append(cb, emptyBytes...)

	return tempbody

}

func Serialize(rawtx []blob) ([]byte, error) {
	length := int64(len(rawtx))

	if length == 0 {
		return nil, fmt.Errorf("Validation failed: Collation Body has to be a non-zero value")
	}
	serialisedData := []byte{}

	for i := int64(0); i < length; i++ {

		blobLength := txblob(serialisedData).length()
		data := rawtx[i].(txblob)
		refinedData := data.serializeBlob()
		serialisedData = append(serialisedData, refinedData...)

		if txblob(serialisedData).length() > collationsizelimit {
			log.Info(fmt.Sprintf("The total number of interfaces added to the collation body are: %d", i))
			serialisedData = serialisedData[:blobLength]
			return serialisedData, nil

		}

	}
	return serialisedData, nil
}

// Collation body deserialised and separated into its respective interfaces

func Deserializebody(collationbody []byte) []blob {

	length := int64(len(collationbody))
	chunksNumber := chunkSize / length
	indicatorByte := make([]byte, 1)
	indicatorByte[0] = 0
	txblobs := []blob{}
	var tempbody txblob

	for i := int64(1); i <= chunksNumber; i++ {

		if reflect.TypeOf(collationbody[(i-1)*chunkSize]) == reflect.TypeOf(indicatorByte) {
			tempbody = append(tempbody, collationbody[((i-1)*chunkSize+1):(i)*chunkSize]...)

		} else {
			terminalIndex := int64(collationbody[(i-1)*chunkSize])
			tempbody = append(tempbody, collationbody[((i-1)*chunkSize+1):((i-1)*chunkSize+2+terminalIndex)]...)
			txblobs = append(txblobs, tempbody)
			tempbody = txblob{}

		}

	}

	return txblobs

}

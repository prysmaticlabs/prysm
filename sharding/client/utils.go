package client

import (
	"fmt"
	"math"
	"reflect"
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
	validateBody() error
	serializeBlob() []byte
}

func (cb txblob) length() int64 {

	return int64(len(cb))
}

/* Validate that the collation body is within its bounds and if
the size of the body is below the limit it is simply appended
till it reaches the required limit */

func (cb txblob) validateBody() error {

	if cb.length() == 0 {
		return fmt.Errorf("Collation Body has to be a non-zero value")
	}

	if cb.length() > totalDatasize {
		return fmt.Errorf("Collation Body is over the size limit")
	}

	return nil
}

func deserializebody(collationbody []byte) []blob {
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

func serialize(rawtx []blob) []byte {
	length := int64(len(rawtx))
	serialisedData := []byte{}

	for i := int64(1); i < length; i++ {
		data := rawtx[length].(txblob)
		refinedData := data.serializeBlob()
		serialisedData = append(serialisedData, refinedData...)
		txblob(serialisedData).validateBody()

	}
	return serialisedData
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

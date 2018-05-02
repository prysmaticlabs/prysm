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

type collationbody []byte

type body interface {
	length() int64
	validateBody() error
	serializeBlob() []byte
}

func (cb collationbody) length() int64 {

	return int64(len(cb))
}

/* Validate that the collation body is within its bounds and if
the size of the body is below the limit it is simply appended
till it reaches the required limit */

func (cb collationbody) validateBody() error {

	if cb.length() == 0 {
		return fmt.Errorf("Collation Body has to be a non-zero value")
	}

	if cb.length() > totalDatasize {
		return fmt.Errorf("Collation Body is over the size limit")
	}

	return nil
}

func deserializeBlob(blob body) []byte {
	deserializedblob := blob.(collationbody)
	length := deserializedblob.length()
	chunksNumber := chunkSize / length
	indicatorByte := make([]byte, 1)
	indicatorByte[0] = 0
	tempbody := []byte{0}

	for i := int64(1); i <= chunksNumber; i++ {

		if reflect.TypeOf(deserializedblob[:(i-1)*chunksNumber]) == reflect.TypeOf(indicatorByte) {

		}

	}

}

// Parse Collation body and modify it accordingly

func (cb collationbody) serializeBlob() []byte {

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

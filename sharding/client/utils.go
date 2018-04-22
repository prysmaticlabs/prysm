package client

import (
	"fmt"
)

type collationbody []byte

var (
	collationsizelimit = int64(2 ^ 20)
	chunkSize          = int64(32)
	indicatorSize      = int64(1)
	numberOfChunks     = collationsizelimit / chunkSize
	chunkDataSize      = chunkSize - indicatorSize
	totalDatasize      = numberOfChunks * chunkDataSize
)

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

// Parse Collation body and modify it accordingly

func (cb collationbody) ParseBlob() {

	terminalLength := cb.length() % chunkDataSize
	chunksNumber := cb.length() / chunkDataSize
	indicatorByte := make([]byte, 1)
	indicatorByte[0] = 0
	var tempbody collationbody

	// Appends empty indicator bytes to non terminal-chunks
	for i := int64(1); i <= chunksNumber; i++ {
		tempbody = append(tempbody, append(indicatorByte, cb[(i-1)*chunkDataSize:i*chunkDataSize]...)...)

	}
	// Appends indicator bytes to terminal-chunks , and if the index of the chunk delimiter is non-zero adds it to the chunk
	if terminalLength != 0 {
		indicatorByte[0] = byte(terminalLength)
		tempbody = append(tempbody, append(indicatorByte, cb[chunksNumber*chunkDataSize:(chunksNumber*chunkDataSize)+(terminalLength+1)]...)...)

	}
	cb = tempbody

	// Pad the collation body with empty bytes until it is equal to 1 Mib
	if cb.length() < collationsizelimit {
		emptyBytes := make([]byte, (collationsizelimit - cb.length()))
		cb = append(cb, emptyBytes...)

	}
}

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

	if cb.length() < totalDatasize {
		x := make([]byte, (totalDatasize - cb.length()))
		cb = append(cb, x...)
		fmt.Printf("%b", x)

	}

	return nil
}

/*
 add
*/

func (cb collationbody) ParseBlob() {
	terminalLength := cb.length() % chunkDataSize
	chunksNumber := cb.length() / chunkDataSize

	if terminalLength != 0 {

	}
}

/*
func createChunks(cb collationbody) {
	validCollation := cb.validateBody
	if err != nil {
		fmt.Errorf("Error %v", err)
	}
	if !validCollation {
		fmt.Errorf("Error %v", err)
	}
	index := int64(len(cb)) / chunkDataSize


	   for i = 0; i < index; i++ {
	   serialisedblob[i*chunksize] = 0
	    for f = 0; f <chunkdatasize ; f++ {
	   serialisedblob[(f+1) + i*chunksize] = collationbody[f + i*chunkdatasize]
	   }
	   serialisedblob[index*chunksize ] = len(collationbody) – index*chunkdatasize
	   }

} */

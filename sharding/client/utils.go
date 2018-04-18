package client

import (
	"fmt"
)

type collationbody []byte

var (
	collationsizelimit = int64(2 * *20)
	chunkSize          = int64(32)
	indicatorSize      = int64(1)
	chunkDataSize      = chunkSize - indicatorSize
)

func (c *collationbody) validateBody() bool {
	return len(c) < 2^20 && len(c) > 0
}

func createChunks(c collationbody) {
	validCollation, err := c.validateBody
	if err != nil {
		fmt.Errorf("Error %v", err)
	}
	if !validCollation {
		fmt.Errorf("Error %v", err)
	}
	index := int64(len(c)) / chunkDataSize

	/*
	   for i = 0; i < index; i++ {
	   serialisedblob[i*chunksize] = 0
	    for f = 0; f <chunkdatasize ; f++ {
	   serialisedblob[(f+1) + i*chunksize] = collationbody[f + i*chunkdatasize]
	   }
	   serialisedblob[index*chunksize ] = len(collationbody) – index*chunkdatasize
	   }*/

}

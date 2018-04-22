package client

import (
	"fmt"
)

type collationbody []byte

var (
	collationsizelimit = int64(2 ^ 20)
	chunkSize          = int64(32)
	indicatorSize      = int64(1)
	chunkDataSize      = chunkSize - indicatorSize
)

/* Validate that the collation body is within its bounds and if
the size of the body is below the limit it is simply appended
till it reaches the required limit */

func (cb collationbody) validateBody() error {

	length := int64(len(cb))

	if length == 0 {
		return fmt.Errorf("Collation Body has to be a non-zero value")
	}

	if length > collationsizelimit {
		return fmt.Errorf("Collation Body is over the size limit")
	}

	if length < collationsizelimit {
		x := make([]byte, (collationsizelimit - length))
		cb = append(cb, x...)
		fmt.Printf("%b", x)

	}

	return nil
}

func main() {

	x := []byte{'h', 'e', 'g', 'g'}
	t := x.validateBody()
	fmt.Printf("%b   %s", x, t)

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

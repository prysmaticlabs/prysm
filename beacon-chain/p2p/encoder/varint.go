package encoder

import (
	"io"

	"github.com/gogo/protobuf/proto"
)

const maxVarintLength = 10

func readVarint(r io.Reader) (uint64, error) {
	b := make([]byte, 0, maxVarintLength)
	for i := 0; i < maxVarintLength; i++ {
		b1 := make([]byte, 1)
		n, err := r.Read(b1)
		if err != nil {
			return 0, err
		}
		if n != 1 {
			panic("n isn't 1") // TODO: do something different than panic.
		}
		b = append(b, b1[0])

		// If most signficant bit is not set, we have reached the end of the Varint.
		if b1[0]&0x80 == 0 {
			break
		}
	}

	vi, n := proto.DecodeVarint(b)
	if n != len(b) {
		panic("n != len(b)") // TODO: do something different than panic.
	}
	return vi, nil
}

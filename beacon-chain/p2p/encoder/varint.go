package encoder

import (
	"errors"
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
			return 0, errors.New("did not read a byte from stream")
		}
		b = append(b, b1[0])

		// If most signficant bit is not set, we have reached the end of the Varint.
		if b1[0]&0x80 == 0 {
			break
		}
	}

	vi, n := proto.DecodeVarint(b)
	if n != len(b) {
		return 0, errors.New("varint did not decode entire byte slice")
	}
	return vi, nil
}

package blobs

import (
	"github.com/protolambda/go-kzg/eth"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
)

type BlobImpl []byte

func (b BlobImpl) At(i int) [32]byte {
	var out [32]byte
	copy(out[:], b[i*32:(i+1)*32-1])
	return out
}

func (b BlobImpl) Len() int {
	return len(b) / 32
}

type BlobsSequenceImpl []*v1.Blob

func (s BlobsSequenceImpl) At(i int) eth.Blob {
	return BlobImpl(s[i].Data)
}

func (s BlobsSequenceImpl) Len() int {
	return len(s)
}

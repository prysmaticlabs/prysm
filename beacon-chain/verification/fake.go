package verification

import "github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"

func BlobSidecarNoop(b blocks.ROBlob) (blocks.VerifiedROBlob, error) {
	return blocks.VerifiedROBlob{ROBlob: b}, nil
}

func BlobSidecarSliceNoop(b []blocks.ROBlob) ([]blocks.VerifiedROBlob, error) {
	vbs := make([]blocks.VerifiedROBlob, len(b))
	for i := range b {
		vbs[i] = blocks.VerifiedROBlob{ROBlob: b[i]}
	}
	return vbs, nil
}

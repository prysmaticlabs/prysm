package logging

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/sirupsen/logrus"
)

// BlobFields extracts a standard set of fields from a BlobSidecar into a logrus.Fields struct
// which can be passed to log.WithFields.
func BlobFields(blob blocks.ROBlob) logrus.Fields {
	return logrus.Fields{
		"slot":           blob.Slot(),
		"proposer_index": blob.ProposerIndex(),
		"block_root":     fmt.Sprintf("%#x", blob.BlockRoot()),
		"parent_root":    fmt.Sprintf("%#x", blob.ParentRoot()),
		"kzg_commitment": fmt.Sprintf("%#x", blob.KzgCommitment),
		"index":          blob.Index,
	}
}

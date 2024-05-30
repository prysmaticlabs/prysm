package merkle_proof

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/electra/merkle_proof"
)

func TestMainnet_Electra_MerkleProof(t *testing.T) {
	t.Skip("TODO: Electra") // These spectests are missing?
	merkle_proof.RunMerkleProofTests(t, "mainnet")
}

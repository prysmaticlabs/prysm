package blocks

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/container/trie"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestComputeBlockBodyFieldRoots_Phase0(t *testing.T) {
	blockBodyPhase0 := hydrateBeaconBlockBody()
	i, err := NewBeaconBlockBody(blockBodyPhase0)
	require.NoError(t, err)

	b := i.(*BeaconBlockBody)

	b.proposerSlashings = []*eth.ProposerSlashing{
		{
			Header_1: &eth.SignedBeaconBlockHeader{
				Header: &eth.BeaconBlockHeader{
					Slot: 1,
				},
			},
			Header_2: &eth.SignedBeaconBlockHeader{
				Header: &eth.BeaconBlockHeader{
					Slot: 2,
				},
			},
		},
	}

	b.attesterSlashings = []*eth.AttesterSlashing{
		{
			Attestation_1: &eth.IndexedAttestation{
				AttestingIndices: []uint64{1, 2, 3},
				Data: &eth.AttestationData{
					Slot: 1,
				},
				Signature: make([]byte, 3),
			},
			Attestation_2: &eth.IndexedAttestation{
				AttestingIndices: []uint64{1, 2, 3},
				Data: &eth.AttestationData{
					Slot: 1,
				},
				Signature: make([]byte, 3),
			},
		},
	}

	b.attestations = []*eth.Attestation{
		{
			AggregationBits: []byte{0b01111100, 0b1},
			Data: &eth.AttestationData{
				Slot: 1,
			},
			Signature: make([]byte, 3),
		},
		{
			AggregationBits: []byte{0b01111000, 0b1},
			Data: &eth.AttestationData{
				Slot: 1,
			},
			Signature: make([]byte, 3),
		},
	}

	b.deposits = []*eth.Deposit{
		{
			Proof: make([][]byte, 2),
			Data: &eth.Deposit_Data{
				PublicKey:             make([]byte, 8),
				WithdrawalCredentials: make([]byte, 8),
				Amount:                1,
				Signature:             make([]byte, 8),
			},
		},
		{
			Proof: make([][]byte, 2),
			Data: &eth.Deposit_Data{
				PublicKey:             make([]byte, 8),
				WithdrawalCredentials: make([]byte, 8),
				Amount:                2,
				Signature:             make([]byte, 8),
			},
		},
	}

	b.voluntaryExits = []*eth.SignedVoluntaryExit{
		{
			Exit: &eth.VoluntaryExit{
				Epoch:          1,
				ValidatorIndex: 1,
			},
			Signature: make([]byte, 8),
		},
		{
			Exit: &eth.VoluntaryExit{
				Epoch:          2,
				ValidatorIndex: 2,
			},
			Signature: make([]byte, 8),
		},
	}

	fieldRoots, err := ComputeBlockBodyFieldRoots(context.Background(), b)
	require.NoError(t, err)
	trie, err := trie.GenerateTrieFromItems(fieldRoots, 3)
	require.NoError(t, err)
	layers := trie.ToProto().GetLayers()

	hash := layers[len(layers)-1].Layer[0]
	require.NoError(t, err)

	correctHash, err := b.HashTreeRoot()
	require.NoError(t, err)

	require.DeepEqual(t, correctHash[:], hash)
}

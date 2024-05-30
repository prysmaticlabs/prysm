package validator

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/operations/attestations/kv"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls/blst"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func Test_computeOnChainAggregate(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.MainnetConfig().Copy()
	cfg.MaxCommitteesPerSlot = 64
	params.OverrideBeaconConfig(cfg)

	key, err := blst.RandKey()
	require.NoError(t, err)
	sig := key.Sign([]byte{'X'})

	data1 := &ethpb.AttestationData{
		Slot:            123,
		CommitteeIndex:  123,
		BeaconBlockRoot: []byte("root"),
		Source: &ethpb.Checkpoint{
			Epoch: 123,
			Root:  []byte("root"),
		},
		Target: &ethpb.Checkpoint{
			Epoch: 123,
			Root:  []byte("root"),
		},
	}
	data2 := &ethpb.AttestationData{
		Slot:            456,
		CommitteeIndex:  456,
		BeaconBlockRoot: []byte("root"),
		Source: &ethpb.Checkpoint{
			Epoch: 456,
			Root:  []byte("root"),
		},
		Target: &ethpb.Checkpoint{
			Epoch: 456,
			Root:  []byte("root"),
		},
	}

	t.Run("single aggregate", func(t *testing.T) {
		att := &ethpb.AttestationElectra{
			AggregationBits: bitfield.Bitlist{0b00011111},
			Data:            data1,
			CommitteeBits:   []byte{0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000001}, // committee index 0
			Signature:       sig.Marshal(),
		}
		id := kv.NewAttestationId(att, [32]byte{'A'})
		aggregates := make(map[kv.AttestationId][]ethpb.Att)
		aggregates[id] = []ethpb.Att{att}

		result, err := computeOnChainAggregate(aggregates)
		require.NoError(t, err)
		require.Equal(t, 1, len(result))
		assert.DeepEqual(t, att.AggregationBits, result[0].GetAggregationBits())
		assert.DeepEqual(t, att.Data, result[0].GetData())
		assert.DeepEqual(t, att.CommitteeBits, result[0].GetCommitteeBitsVal())
	})
	t.Run("all aggregates for one root", func(t *testing.T) {
		att1 := &ethpb.AttestationElectra{
			AggregationBits: bitfield.Bitlist{0b00010011}, // aggregation bits 0,1
			Data:            data1,
			CommitteeBits:   []byte{0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000001}, // committee index 0
			Signature:       sig.Marshal(),
		}
		att2 := &ethpb.AttestationElectra{
			AggregationBits: bitfield.Bitlist{0b00010011}, // aggregation bits 0,1
			Data:            data1,
			CommitteeBits:   []byte{0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000010}, // committee index 1
			Signature:       sig.Marshal(),
		}
		id1 := kv.NewAttestationId(att1, [32]byte{'A'})
		id2 := kv.NewAttestationId(att2, [32]byte{'A'})
		aggregates := make(map[kv.AttestationId][]ethpb.Att)
		aggregates[id1] = []ethpb.Att{att1}
		aggregates[id2] = []ethpb.Att{att2}

		result, err := computeOnChainAggregate(aggregates)
		require.NoError(t, err)
		require.Equal(t, 1, len(result))
		assert.DeepEqual(t, bitfield.Bitlist{0b00110011, 0b00000001}, result[0].GetAggregationBits())
		assert.DeepEqual(t, data1, result[0].GetData())
		assert.DeepEqual(t, bitfield.Bitvector64([]byte{0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000011}), result[0].GetCommitteeBitsVal())
	})
	t.Run("aggregates for multiple roots", func(t *testing.T) {
		att1 := &ethpb.AttestationElectra{
			AggregationBits: bitfield.Bitlist{0b00010011}, // aggregation bits 0,1
			Data:            data1,
			CommitteeBits:   []byte{0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000001}, // committee index 0
			Signature:       sig.Marshal(),
		}
		att2 := &ethpb.AttestationElectra{
			AggregationBits: bitfield.Bitlist{0b00010011}, // aggregation bits 0,1
			Data:            data1,
			CommitteeBits:   []byte{0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000010}, // committee index 1
			Signature:       sig.Marshal(),
		}
		att3 := &ethpb.AttestationElectra{
			AggregationBits: bitfield.Bitlist{0b00011001}, // aggregation bits 0,3
			Data:            data2,
			CommitteeBits:   []byte{0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000001}, // committee index 0
			Signature:       sig.Marshal(),
		}
		att4 := &ethpb.AttestationElectra{
			AggregationBits: bitfield.Bitlist{0b00010010}, // aggregation bits 1
			Data:            data2,
			CommitteeBits:   []byte{0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000010}, // committee index 1
			Signature:       sig.Marshal(),
		}
		id1 := kv.NewAttestationId(att1, [32]byte{'A'})
		id2 := kv.NewAttestationId(att2, [32]byte{'A'})
		id3 := kv.NewAttestationId(att3, [32]byte{'B'})
		id4 := kv.NewAttestationId(att4, [32]byte{'B'})
		aggregates := make(map[kv.AttestationId][]ethpb.Att)
		aggregates[id1] = []ethpb.Att{att1}
		aggregates[id2] = []ethpb.Att{att2}
		aggregates[id3] = []ethpb.Att{att3}
		aggregates[id4] = []ethpb.Att{att4}

		result, err := computeOnChainAggregate(aggregates)
		require.NoError(t, err)
		require.Equal(t, 2, len(result))
		expectedCommitteeBits := bitfield.Bitvector64([]byte{0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000011})

		expectedAggBits := bitfield.Bitlist{0b00110011, 0b00000001}
		expectedData := data1
		found := false
		for _, a := range result {
			if reflect.DeepEqual(expectedAggBits, a.GetAggregationBits()) && reflect.DeepEqual(expectedData, a.GetData()) && reflect.DeepEqual(expectedCommitteeBits, a.GetCommitteeBitsVal()) {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected aggregate not found")
		}

		expectedAggBits = bitfield.Bitlist{0b00101001, 0b00000001}
		expectedData = data2
		found = false
		for _, a := range result {
			if reflect.DeepEqual(expectedAggBits, a.GetAggregationBits()) && reflect.DeepEqual(expectedData, a.GetData()) && reflect.DeepEqual(expectedCommitteeBits, a.GetCommitteeBitsVal()) {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected aggregate not found")
		}
	})
	t.Run("duplicate committee index is OK", func(t *testing.T) {

	})
}

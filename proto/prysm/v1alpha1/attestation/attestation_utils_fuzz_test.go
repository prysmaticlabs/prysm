package attestation_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func FuzzConvertToIndexed(f *testing.F) {
	att := util.HydrateAttestation(&eth.Attestation{})
	d, err := att.MarshalSSZ()
	require.NoError(f, err)

	f.Add(d, []byte(att.AggregationBits))

	f.Fuzz(func(t *testing.T, attRaw []byte, committeeBitlistRaw []byte) {
		att := &eth.Attestation{}
		err := att.UnmarshalSSZ(attRaw)
		if err != nil {
			return
		}
		cb := bitfield.Bitlist(committeeBitlistRaw)
		var committee []types.ValidatorIndex
		for _, idx := range cb.BitIndices() {
			committee = append(committee, types.ValidatorIndex(idx))
		}
		_, err = attestation.ConvertToIndexed(context.TODO(), att, committee)
		_ = err
	})
}

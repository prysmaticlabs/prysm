package attestations_test

import (
	"testing"

	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation/aggregation/attestations"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func FuzzAggregate(f *testing.F) {
	att := util.HydrateAttestation(&eth.Attestation{})
	data, err := att.MarshalSSZ()
	require.NoError(f, err)
	f.Add(data, data)

	f.Fuzz(func(t *testing.T, a []byte, b []byte) {
		aa := &eth.Attestation{}
		bb := &eth.Attestation{}
		if err := aa.UnmarshalSSZ(a); err != nil {
			return
		}
		if err := bb.UnmarshalSSZ(b); err != nil {
			return
		}
		_, err := attestations.Aggregate([]*eth.Attestation{aa, bb})
		if err != nil {
			return
		}
	})
}

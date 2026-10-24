package operations

import (
	"context"
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/eip8453"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

// RunAttestationTest executes "operations/attestation" tests.
func RunAttestationTest(t *testing.T, config string) {
	runOperationTest(t, config, "attestation", func(b []byte) (*ethpb.BeaconBlockBodyEIP8453, error) {
		att := &ethpb.AttestationEIP8453{}
		if err := att.UnmarshalSSZ(b); err != nil {
			return nil, err
		}
		return &ethpb.BeaconBlockBodyEIP8453{Attestations: []*ethpb.AttestationEIP8453{att}}, nil
	}, func(ctx context.Context, s state.BeaconState, body interfaces.ReadOnlyBeaconBlockBody) error {
		atts, err := body.AttestationsEIP8453()
		if err != nil {
			return err
		}
		return eip8453.ProcessAttestation(ctx, s, atts[0])
	})
}

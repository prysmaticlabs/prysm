package operations

import (
	"context"
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/eip8453"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

// RunAttesterSlashingTest executes "operations/attester_slashing" tests.
func RunAttesterSlashingTest(t *testing.T, config string) {
	runOperationTest(t, config, "attester_slashing", func(b []byte) (*ethpb.BeaconBlockBodyEIP8453, error) {
		slashing := &ethpb.AttesterSlashingEIP8453{}
		if err := slashing.UnmarshalSSZ(b); err != nil {
			return nil, err
		}
		return &ethpb.BeaconBlockBodyEIP8453{AttesterSlashings: []*ethpb.AttesterSlashingEIP8453{slashing}}, nil
	}, func(ctx context.Context, s state.BeaconState, body interfaces.ReadOnlyBeaconBlockBody) error {
		slashings, err := body.AttesterSlashingsEIP8453()
		if err != nil {
			return err
		}
		return eip8453.ProcessAttesterSlashing(ctx, s, slashings[0])
	})
}

// RunLegacyAttesterSlashingTest executes "operations/legacy_attester_slashing" tests.
func RunLegacyAttesterSlashingTest(t *testing.T, config string) {
	runOperationTest(t, config, "legacy_attester_slashing", func(b []byte) (*ethpb.BeaconBlockBodyEIP8453, error) {
		slashing := &ethpb.AttesterSlashingGloas{}
		if err := slashing.UnmarshalSSZ(b); err != nil {
			return nil, err
		}
		return &ethpb.BeaconBlockBodyEIP8453{LegacyAttesterSlashings: []*ethpb.AttesterSlashingGloas{slashing}}, nil
	}, func(ctx context.Context, s state.BeaconState, body interfaces.ReadOnlyBeaconBlockBody) error {
		slashings, err := body.LegacyAttesterSlashings()
		if err != nil {
			return err
		}
		return eip8453.ProcessLegacyAttesterSlashing(ctx, s, slashings[0])
	})
}

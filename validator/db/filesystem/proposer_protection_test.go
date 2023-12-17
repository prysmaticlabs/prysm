package filesystem

import (
	"context"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/validator/db/kv"
)

func TestStore_ProposalHistoryForPubKey(t *testing.T) {
	var slot uint64 = 42
	ctx := context.Background()

	for _, tt := range []struct {
		name                        string
		validatorSlashingProtection *ValidatorSlashingProtection
		expectedProposals           []*kv.Proposal
	}{
		{
			name:                        "validatorSlashingProtection is nil",
			validatorSlashingProtection: nil,
			expectedProposals:           []*kv.Proposal{},
		},
		{
			name:                        "validatorSlashingProtection.LatestSignedBlockSlot is nil",
			validatorSlashingProtection: &ValidatorSlashingProtection{LatestSignedBlockSlot: nil},
			expectedProposals:           []*kv.Proposal{},
		},
		{
			name:                        "validatorSlashingProtection.LatestSignedBlockSlot is something",
			validatorSlashingProtection: &ValidatorSlashingProtection{LatestSignedBlockSlot: &slot},
			expectedProposals: []*kv.Proposal{
				{
					Slot: primitives.Slot(slot),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Get a database path.
			databasePath := t.TempDir()

			// Create a public key.
			pubkey := getPubKeys(t, 1)[0]

			// Create a new store.
			store, err := NewStore(databasePath, nil)
			require.NoError(t, err, "NewStore should not return an error")

			// Set the validator slashing protection.
			err = store.saveValidatorSlashingProtection(pubkey, tt.validatorSlashingProtection)
			require.NoError(t, err, "saveValidatorSlashingProtection should not return an error")

			// Get the proposal history for the public key.
			actualProposals, err := store.ProposalHistoryForPubKey(ctx, pubkey)
			require.NoError(t, err, "ProposalHistoryForPubKey should not return an error")
			require.DeepEqual(t, tt.expectedProposals, actualProposals, "ProposalHistoryForPubKey should return the expected proposals")
		})
	}
}

func TestStore_SaveProposalHistoryForSlot(t *testing.T) {
	var (
		slot41 uint64 = 41
		slot42 uint64 = 42
		slot43 uint64 = 43
	)

	ctx := context.Background()

	for _, tt := range []struct {
		name                                string
		initialValidatorSlashingProtection  *ValidatorSlashingProtection
		slot                                uint64
		expectedValidatorSlashingProtection ValidatorSlashingProtection
		expectedError                       string
	}{
		{
			name:                                "validatorSlashingProtection is nil",
			initialValidatorSlashingProtection:  nil,
			slot:                                slot42,
			expectedValidatorSlashingProtection: ValidatorSlashingProtection{LatestSignedBlockSlot: &slot42},
			expectedError:                       "",
		},
		{
			name:                                "validatorSlashingProtection.LatestSignedBlockSlot is nil",
			initialValidatorSlashingProtection:  &ValidatorSlashingProtection{LatestSignedBlockSlot: nil},
			slot:                                slot42,
			expectedValidatorSlashingProtection: ValidatorSlashingProtection{LatestSignedBlockSlot: &slot42},
			expectedError:                       "",
		},
		{
			name:                                "validatorSlashingProtection.LatestSignedBlockSlot is lower than the incoming slot",
			initialValidatorSlashingProtection:  &ValidatorSlashingProtection{LatestSignedBlockSlot: &slot42},
			slot:                                slot41,
			expectedValidatorSlashingProtection: ValidatorSlashingProtection{LatestSignedBlockSlot: &slot42},
			expectedError:                       "could not sign proposal with slot lower than or equal to recorded slot",
		},
		{
			name:                                "validatorSlashingProtection.LatestSignedBlockSlot is equal to the incoming slot",
			initialValidatorSlashingProtection:  &ValidatorSlashingProtection{LatestSignedBlockSlot: &slot42},
			slot:                                slot42,
			expectedValidatorSlashingProtection: ValidatorSlashingProtection{LatestSignedBlockSlot: &slot42},
			expectedError:                       "could not sign proposal with slot lower than or equal to recorded slot",
		},
		{
			name:                                "validatorSlashingProtection.LatestSignedBlockSlot is higher to the incoming slot",
			initialValidatorSlashingProtection:  &ValidatorSlashingProtection{LatestSignedBlockSlot: &slot42},
			slot:                                slot43,
			expectedValidatorSlashingProtection: ValidatorSlashingProtection{LatestSignedBlockSlot: &slot43},
			expectedError:                       "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Get a database path.
			databasePath := t.TempDir()

			// Create a public key.
			pubkey := getPubKeys(t, 1)[0]

			// Create a new store.
			store, err := NewStore(databasePath, nil)
			require.NoError(t, err, "NewStore should not return an error")

			// Set the initial validator slashing protection.
			err = store.saveValidatorSlashingProtection(pubkey, tt.initialValidatorSlashingProtection)
			require.NoError(t, err, "saveValidatorSlashingProtection should not return an error")

			// Attempt to save the proposal history for the public key.
			err = store.SaveProposalHistoryForSlot(ctx, pubkey, primitives.Slot(tt.slot), nil)
			if len(tt.expectedError) > 0 {
				require.ErrorContains(t, tt.expectedError, err, "validatorSlashingProtection should return the expected error")
			} else {
				require.NoError(t, err, "SaveProposalHistoryForSlot should not return an error")
			}

			// Get the final validator slashing protection.
			actualValidatorSlashingProtection, err := store.validatorSlashingProtection(pubkey)
			require.NoError(t, err, "validatorSlashingProtection should not return an error")

			// Check the proposal history.
			require.DeepEqual(t, tt.expectedValidatorSlashingProtection, *actualValidatorSlashingProtection, "validatorSlashingProtection should be the expected one")
		})
	}
}

func TestStore_ProposedPublicKeys(t *testing.T) {
	// We get a database path
	databasePath := t.TempDir()

	// We create some pubkeys
	pubkeys := getPubKeys(t, 5)

	// We create a new store
	s, err := NewStore(databasePath, &Config{PubKeys: pubkeys})
	require.NoError(t, err, "NewStore should not return an error")

	// We check the public keys
	expected := pubkeys
	actual, err := s.ProposedPublicKeys(context.Background())
	require.NoError(t, err, "publicKeys should not return an error")

	// We cannot compare the slices directly because the order is not guaranteed,
	// so we compare sets instead.
	expectedSet := make(map[[fieldparams.BLSPubkeyLength]byte]bool)
	for _, pubkey := range expected {
		expectedSet[pubkey] = true
	}

	actualSet := make(map[[fieldparams.BLSPubkeyLength]byte]bool)
	for _, pubkey := range actual {
		actualSet[pubkey] = true
	}

	require.DeepEqual(t, expectedSet, actualSet)
}

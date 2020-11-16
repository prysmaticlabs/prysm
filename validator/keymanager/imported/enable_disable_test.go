package imported

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestKeymanager_DisableAccounts(t *testing.T) {
	numKeys := 5
	randomPublicKeys := make([][]byte, numKeys)
	for i := 0; i < numKeys; i++ {
		key, err := bls.RandKey()
		require.NoError(t, err)
		randomPublicKeys[i] = key.PublicKey().Marshal()
	}
	tests := []struct {
		name                 string
		existingDisabledKeys [][]byte
		keysToDisable        [][]byte
		expectedDisabledKeys [][]byte
		wantErr              bool
	}{
		{
			name:                 "Trying to disable already disabled keys fails silently",
			existingDisabledKeys: randomPublicKeys,
			keysToDisable:        randomPublicKeys,
			wantErr:              false,
			expectedDisabledKeys: randomPublicKeys,
		},
		{
			name:                 "Trying to disable a subset of keys works",
			existingDisabledKeys: randomPublicKeys[0:2],
			keysToDisable:        randomPublicKeys[2:],
			wantErr:              false,
			expectedDisabledKeys: randomPublicKeys,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := &Keymanager{
				disabledPublicKeys: tt.existingDisabledKeys,
			}
			ctx := context.Background()
			if err := dr.DisableAccounts(ctx, tt.keysToDisable); (err != nil) != tt.wantErr {
				t.Errorf("DisableAccounts() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, len(tt.expectedDisabledKeys), len(dr.disabledPublicKeys))
			//assert.DeepEqual(t, tt.expectedDisabledKeys, dr.disabledPublicKeys)
		})
	}
}

func TestKeymanager_EnableAccounts(t *testing.T) {
	numKeys := 5
	randomPublicKeys := make([][]byte, numKeys)
	for i := 0; i < numKeys; i++ {
		key, err := bls.RandKey()
		require.NoError(t, err)
		randomPublicKeys[i] = key.PublicKey().Marshal()
	}

	tests := []struct {
		name                 string
		existingDisabledKeys [][]byte
		keysToEnable         [][]byte
		wantErr              bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := &Keymanager{
				disabledPublicKeys: tt.existingDisabledKeys,
			}
			ctx := context.Background()
			if err := dr.EnableAccounts(ctx, tt.keysToEnable); (err != nil) != tt.wantErr {
				t.Errorf("EnableAccounts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

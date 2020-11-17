package imported

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
		{
			name:                 "Nil input keys to disable returns error",
			existingDisabledKeys: randomPublicKeys,
			keysToDisable:        nil,
			wantErr:              true,
		},
		{
			name:                 "No input keys to disable returns error",
			existingDisabledKeys: randomPublicKeys,
			keysToDisable:        make([][]byte, 0),
			wantErr:              true,
		},
		{
			name:                 "No existing disabled keys updates after disabling",
			existingDisabledKeys: make([][]byte, 0),
			keysToDisable:        randomPublicKeys,
			expectedDisabledKeys: randomPublicKeys,
		},
		{
			name:                 "Disjoint sets of already disabled + newly disabled leads to whole set",
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
			if !tt.wantErr {
				wanted := make(map[[48]byte]bool)
				for _, pubKey := range tt.expectedDisabledKeys {
					wanted[bytesutil.ToBytes48(pubKey)] = true
				}
				for _, pubKey := range dr.disabledPublicKeys {
					if _, ok := wanted[bytesutil.ToBytes48(pubKey)]; !ok {
						t.Errorf("Expected %#x in disabled keys, but not found", pubKey)
					}
				}
			}
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

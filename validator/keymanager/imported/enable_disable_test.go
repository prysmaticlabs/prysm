package imported

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/testing"
)

func TestKeymanager_DisableAccounts(t *testing.T) {
	numKeys := 5
	randomPrivateKeys := make([][]byte, numKeys)
	randomPublicKeys := make([][]byte, numKeys)
	for i := 0; i < numKeys; i++ {
		key, err := bls.RandKey()
		require.NoError(t, err)
		randomPrivateKeys[i] = key.Marshal()
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
			wallet := &mock.Wallet{
				Files: make(map[string]map[string][]byte),
			}
			disabledPubKeys := make(map[[48]byte]bool)
			for _, pubKey := range tt.existingDisabledKeys {
				disabledPubKeys[bytesutil.ToBytes48(pubKey)] = true
			}
			dr := &Keymanager{
				disabledPublicKeys: disabledPubKeys,
				wallet:             wallet,
			}
			// First we write the accounts store file.
			ctx := context.Background()
			store, err := dr.createAccountsKeystore(ctx, randomPrivateKeys, randomPublicKeys)
			require.NoError(t, err)
			existingDisabledKeysStr := make([]string, len(tt.existingDisabledKeys))
			for i := 0; i < len(tt.existingDisabledKeys); i++ {
				existingDisabledKeysStr[i] = fmt.Sprintf("%x", tt.existingDisabledKeys[i])
			}
			store.DisabledPublicKeys = existingDisabledKeysStr
			encoded, err := json.Marshal(store)
			require.NoError(t, err)
			err = dr.wallet.WriteFileAtPath(ctx, AccountsPath, accountsKeystoreFileName, encoded)
			require.NoError(t, err)

			if err := dr.DisableAccounts(ctx, tt.keysToDisable); (err != nil) != tt.wantErr {
				t.Errorf("DisableAccounts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				wanted := make(map[[48]byte]bool)
				for _, pubKey := range tt.expectedDisabledKeys {
					wanted[bytesutil.ToBytes48(pubKey)] = true
				}
				// We verify that the updated disabled keys are reflected on disk as well.
				encoded, err := wallet.ReadFileAtPath(ctx, AccountsPath, accountsKeystoreFileName)
				require.NoError(t, err)
				keystore := &accountsKeystoreRepresentation{}
				require.NoError(t, json.Unmarshal(encoded, keystore))

				require.Equal(t, len(wanted), len(keystore.DisabledPublicKeys))
				for _, pubKey := range keystore.DisabledPublicKeys {
					pubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(pubKey, "0x"))
					require.NoError(t, err)
					if _, ok := wanted[bytesutil.ToBytes48(pubKeyBytes)]; !ok {
						t.Errorf("Expected %#x in disabled keys, but not found", pubKeyBytes)
					}
				}
			}
		})
	}
}

func TestKeymanager_EnableAccounts(t *testing.T) {
	numKeys := 5
	randomPrivateKeys := make([][]byte, numKeys)
	randomPublicKeys := make([][]byte, numKeys)
	for i := 0; i < numKeys; i++ {
		key, err := bls.RandKey()
		require.NoError(t, err)
		randomPrivateKeys[i] = key.Marshal()
		randomPublicKeys[i] = key.PublicKey().Marshal()
	}
	tests := []struct {
		name                 string
		existingDisabledKeys [][]byte
		keysToEnable         [][]byte
		expectedDisabledKeys [][]byte
		wantErr              bool
	}{
		{
			name:                 "Trying to enable already enabled keys fails silently",
			existingDisabledKeys: make([][]byte, 0),
			keysToEnable:         randomPublicKeys,
			wantErr:              false,
			expectedDisabledKeys: nil,
		},
		{
			name:                 "Trying to enable a subset of keys works",
			existingDisabledKeys: randomPublicKeys[0:2],
			keysToEnable:         randomPublicKeys[0:1],
			wantErr:              false,
			expectedDisabledKeys: randomPublicKeys[1:2],
		},
		{
			name:                 "Nil input keys to enable returns error",
			existingDisabledKeys: randomPublicKeys,
			keysToEnable:         nil,
			wantErr:              true,
		},
		{
			name:                 "No input keys to enable returns error",
			existingDisabledKeys: randomPublicKeys,
			keysToEnable:         make([][]byte, 0),
			wantErr:              true,
		},
		{
			name:                 "No existing enabled keys updates after enablng",
			existingDisabledKeys: randomPublicKeys,
			keysToEnable:         randomPublicKeys,
			expectedDisabledKeys: make([][]byte, 0),
		},
		{
			name:                 "Disjoint sets of already enabled + newly enabled leads to whole set",
			existingDisabledKeys: randomPublicKeys[0:2],
			keysToEnable:         randomPublicKeys[0:2],
			wantErr:              false,
			expectedDisabledKeys: make([][]byte, 0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wallet := &mock.Wallet{
				Files: make(map[string]map[string][]byte),
			}
			disabledPubKeys := make(map[[48]byte]bool)
			for _, pubKey := range tt.existingDisabledKeys {
				disabledPubKeys[bytesutil.ToBytes48(pubKey)] = true
			}
			dr := &Keymanager{
				disabledPublicKeys: disabledPubKeys,
				wallet:             wallet,
			}
			// First we write the accounts store file.
			ctx := context.Background()
			store, err := dr.createAccountsKeystore(ctx, randomPrivateKeys, randomPublicKeys)
			require.NoError(t, err)
			existingDisabledKeysStr := make([]string, len(tt.existingDisabledKeys))
			for i := 0; i < len(tt.existingDisabledKeys); i++ {
				existingDisabledKeysStr[i] = fmt.Sprintf("%x", tt.existingDisabledKeys[i])
			}
			store.DisabledPublicKeys = existingDisabledKeysStr
			encoded, err := json.Marshal(store)
			require.NoError(t, err)
			err = dr.wallet.WriteFileAtPath(ctx, AccountsPath, accountsKeystoreFileName, encoded)
			require.NoError(t, err)

			if err := dr.EnableAccounts(ctx, tt.keysToEnable); (err != nil) != tt.wantErr {
				t.Errorf("EnableAccounts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				wanted := make(map[[48]byte]bool)
				for _, pubKey := range tt.expectedDisabledKeys {
					wanted[bytesutil.ToBytes48(pubKey)] = true
				}
				for pubKey := range dr.disabledPublicKeys {
					if _, ok := wanted[pubKey]; !ok {
						t.Errorf("Expected %#x in disabled keys, but not found", pubKey)
					}
				}
				// We verify that the updated disabled keys are reflected on disk as well.
				encoded, err := wallet.ReadFileAtPath(ctx, AccountsPath, accountsKeystoreFileName)
				require.NoError(t, err)
				keystore := &accountsKeystoreRepresentation{}
				require.NoError(t, json.Unmarshal(encoded, keystore))

				require.Equal(t, len(wanted), len(keystore.DisabledPublicKeys))
				for _, pubKey := range keystore.DisabledPublicKeys {
					pubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(pubKey, "0x"))
					require.NoError(t, err)
					if _, ok := wanted[bytesutil.ToBytes48(pubKeyBytes)]; !ok {
						t.Errorf("Expected %#x in disabled keys, but not found", pubKeyBytes)
					}
				}
			}
		})
	}
}

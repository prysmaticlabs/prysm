package v2

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
)

func TestCreateAccount_Derived(t *testing.T) {
	walletDir, passwordsDir, passwordFile := setupWalletAndPasswordsDir(t)
	numAccounts := int64(5)
	cliCtx := setupWalletCtx(t, &testWalletConfig{
		walletDir:      walletDir,
		passwordsDir:   passwordsDir,
		passwordFile:   passwordFile,
		keymanagerKind: v2keymanager.Derived,
		numAccounts:    numAccounts,
	})

	// We attempt to create the wallet.
	require.NoError(t, CreateWallet(cliCtx))

	// We attempt to open the newly created wallet.
	ctx := context.Background()
	wallet, err := OpenWallet(cliCtx)
	assert.NoError(t, err)

	// We read the keymanager config for the newly created wallet.
	encoded, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
	assert.NoError(t, err)
	cfg, err := derived.UnmarshalConfigFile(encoded)
	assert.NoError(t, err)

	// We assert the created configuration was as desired.
	assert.DeepEqual(t, derived.DefaultConfig(), cfg)

	require.NoError(t, CreateAccount(cliCtx))

	keymanager, err := wallet.InitializeKeymanager(ctx, true)
	require.NoError(t, err)
	km, ok := keymanager.(*derived.Keymanager)
	if !ok {
		t.Fatal("not a derived keymanager")
	}
	names, err := km.ValidatingAccountNames(ctx)
	assert.NoError(t, err)
	require.Equal(t, len(names), int(numAccounts))
}

func Test_validatePasswordInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "no numbers nor special characters",
			input:   "abcdefghijklmnopqrs",
			wantErr: true,
		},
		{
			name:    "number and letters but no special characters",
			input:   "abcdefghijklmnopqrs2020",
			wantErr: true,
		},
		{
			name:    "numbers, letters, special characters, but too short",
			input:   "abc2$",
			wantErr: true,
		},
		{
			name:    "proper length and strong password",
			input:   "%Str0ngpassword32kjAjsd22020$%",
			wantErr: false,
		},
		{
			name:    "password format correct but weak entropy score",
			input:   "aaaaaaa1$",
			wantErr: true,
		},
		{
			name:    "Unicode strings separated by a space character",
			input:   "x*329293@aAJSD i22903saj",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePasswordInput(tt.input); (err != nil) != tt.wantErr {
				t.Errorf("validatePasswordInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_isValidUnicode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "Regular alphanumeric",
			input: "Someone23xx",
			want:  true,
		},
		{
			name:  "Unicode strings separated by a space character",
			input: "x*329293@aAJSD i22903saj",
			want:  false,
		},
		{
			name:  "Japanese",
			input: "僕は絵お見るのが好きです",
			want:  true,
		},
		{
			name:  "Other foreign",
			input: "Etérium",
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidUnicode(tt.input); got != tt.want {
				t.Errorf("isValidUnicode() = %v, want %v", got, tt.want)
			}
		})
	}
}

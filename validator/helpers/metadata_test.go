package helpers

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
	dbtest "github.com/prysmaticlabs/prysm/v5/validator/db/testing"
	"github.com/prysmaticlabs/prysm/v5/validator/slashing-protection-history/format"
)

func Test_validateMetadata(t *testing.T) {
	goodRoot := [32]byte{1}
	goodStr := make([]byte, hex.EncodedLen(len(goodRoot)))
	hex.Encode(goodStr, goodRoot[:])
	tests := []struct {
		name                    string
		interchangeJSON         *format.EIPSlashingProtectionFormat
		dbGenesisValidatorsRoot []byte
		wantErr                 bool
		wantFatal               string
	}{
		{
			name: "Incorrect version for EIP format should fail",
			interchangeJSON: &format.EIPSlashingProtectionFormat{
				Metadata: struct {
					InterchangeFormatVersion string `json:"interchange_format_version"`
					GenesisValidatorsRoot    string `json:"genesis_validators_root"`
				}{
					InterchangeFormatVersion: "1",
					GenesisValidatorsRoot:    string(goodStr),
				},
			},
			wantErr: true,
		},
		{
			name: "Junk data for version should fail",
			interchangeJSON: &format.EIPSlashingProtectionFormat{
				Metadata: struct {
					InterchangeFormatVersion string `json:"interchange_format_version"`
					GenesisValidatorsRoot    string `json:"genesis_validators_root"`
				}{
					InterchangeFormatVersion: "asdljas$d",
					GenesisValidatorsRoot:    string(goodStr),
				},
			},
			wantErr: true,
		},
		{
			name: "Proper version field should pass",
			interchangeJSON: &format.EIPSlashingProtectionFormat{
				Metadata: struct {
					InterchangeFormatVersion string `json:"interchange_format_version"`
					GenesisValidatorsRoot    string `json:"genesis_validators_root"`
				}{
					InterchangeFormatVersion: format.InterchangeFormatVersion,
					GenesisValidatorsRoot:    string(goodStr),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		for _, isSlashingProtectionMinimal := range [...]bool{false, true} {
			t.Run(fmt.Sprintf("%s/isSlashingProtectionMinimal=%v", tt.name, isSlashingProtectionMinimal), func(t *testing.T) {
				validatorDB := dbtest.SetupDB(t, nil, isSlashingProtectionMinimal)
				ctx := context.Background()
				if err := ValidateMetadata(ctx, validatorDB, tt.interchangeJSON); (err != nil) != tt.wantErr {
					t.Errorf("validateMetadata() error = %v, wantErr %v", err, tt.wantErr)
				}

			})
		}
	}
}

func Test_validateMetadataGenesisValidatorsRoot(t *testing.T) {
	goodRoot := [32]byte{1}
	goodStr := make([]byte, hex.EncodedLen(len(goodRoot)))
	hex.Encode(goodStr, goodRoot[:])
	secondRoot := [32]byte{2}
	secondStr := make([]byte, hex.EncodedLen(len(secondRoot)))
	hex.Encode(secondStr, secondRoot[:])

	tests := []struct {
		name                    string
		interchangeJSON         *format.EIPSlashingProtectionFormat
		dbGenesisValidatorsRoot []byte
		wantErr                 bool
	}{
		{
			name: "Same genesis roots should not fail",
			interchangeJSON: &format.EIPSlashingProtectionFormat{
				Metadata: struct {
					InterchangeFormatVersion string `json:"interchange_format_version"`
					GenesisValidatorsRoot    string `json:"genesis_validators_root"`
				}{
					InterchangeFormatVersion: format.InterchangeFormatVersion,
					GenesisValidatorsRoot:    string(goodStr),
				},
			},
			dbGenesisValidatorsRoot: goodRoot[:],
			wantErr:                 false,
		},
		{
			name: "Different genesis roots should not fail",
			interchangeJSON: &format.EIPSlashingProtectionFormat{
				Metadata: struct {
					InterchangeFormatVersion string `json:"interchange_format_version"`
					GenesisValidatorsRoot    string `json:"genesis_validators_root"`
				}{
					InterchangeFormatVersion: format.InterchangeFormatVersion,
					GenesisValidatorsRoot:    string(secondStr),
				},
			},
			dbGenesisValidatorsRoot: goodRoot[:],
			wantErr:                 true,
		},
	}
	for _, tt := range tests {
		for _, isSlashingProtectionMinimal := range [...]bool{false, true} {
			t.Run(fmt.Sprintf("%s/isSlashingProtectionMinimal=%v", tt.name, isSlashingProtectionMinimal), func(t *testing.T) {
				validatorDB := dbtest.SetupDB(t, nil, isSlashingProtectionMinimal)
				ctx := context.Background()
				require.NoError(t, validatorDB.SaveGenesisValidatorsRoot(ctx, tt.dbGenesisValidatorsRoot))
				err := ValidateMetadata(ctx, validatorDB, tt.interchangeJSON)
				if tt.wantErr {
					require.ErrorContains(t, "genesis validators root doesn't match the one that is stored", err)
				} else {
					require.NoError(t, err)
				}

			})
		}
	}
}

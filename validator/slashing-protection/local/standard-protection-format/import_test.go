package interchangeformat_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
	"github.com/prysmaticlabs/prysm/validator/slashing-protection/local/standard-protection-format"
	spTest "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/testing"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestStore_ImportInterchangeData_BadJSON(t *testing.T) {
	ctx := context.Background()
	validatorDB := dbtest.SetupDB(t, nil)

	buf := bytes.NewBuffer([]byte("helloworld"))
	err := interchangeformat.ImportStandardProtectionJSON(ctx, validatorDB, buf)
	require.ErrorContains(t, "could not unmarshal slashing protection JSON file", err)
}

func TestStore_ImportInterchangeData_NilData_FailsSilently(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	validatorDB := dbtest.SetupDB(t, nil)

	interchangeJSON := &interchangeformat.EIPSlashingProtectionFormat{}
	encoded, err := json.Marshal(interchangeJSON)
	require.NoError(t, err)

	buf := bytes.NewBuffer(encoded)
	err = interchangeformat.ImportStandardProtectionJSON(ctx, validatorDB, buf)
	require.NoError(t, err)
	require.LogsContain(t, hook, "No slashing protection data to import")
}

func TestStore_ImportInterchangeData_BadFormat_PreventsDBWrites(t *testing.T) {
	ctx := context.Background()
	numValidators := 10
	publicKeys := spTest.CreateRandomPubKeys(t, numValidators)
	validatorDB := dbtest.SetupDB(t, publicKeys)

	// First we setup some mock attesting and proposal histories and create a mock
	// standard slashing protection format JSON struct.
	attestingHistory, proposalHistory := spTest.MockAttestingAndProposalHistories(t, numValidators)
	standardProtectionFormat := spTest.MockSlashingProtectionJSON(t, publicKeys, attestingHistory, proposalHistory)

	// We replace a slot of one of the blocks with junk data.
	standardProtectionFormat.Data[0].SignedBlocks[0].Slot = "BadSlot"

	// We encode the standard slashing protection struct into a JSON format.
	blob, err := json.Marshal(standardProtectionFormat)
	require.NoError(t, err)
	buf := bytes.NewBuffer(blob)

	// Next, we attempt to import it into our validator database and check that
	// we obtain an error during the import process.
	err = interchangeformat.ImportStandardProtectionJSON(ctx, validatorDB, buf)
	assert.NotNil(t, err)

	// Next, we attempt to retrieve the attesting and proposals histories from our database and
	// verify nothing was saved to the DB. If there is an error in the import process, we need to make
	// sure writing is an atomic operation: either the import succeeds and saves the slashing protection
	// data to our DB, or it does not.
	receivedAttestingHistory, err := validatorDB.AttestationHistoryForPubKeysV2(ctx, publicKeys)
	require.NoError(t, err)
	for i := 0; i < len(publicKeys); i++ {
		defaultAttestingHistory := kv.NewAttestationHistoryArray(0)
		require.DeepEqual(
			t,
			defaultAttestingHistory,
			receivedAttestingHistory[publicKeys[i]],
			"Imported attestation protection history is different than the empty default",
		)
		proposals := proposalHistory[i].Proposals
		for _, proposal := range proposals {
			receivedProposalSigningRoot, err := validatorDB.ProposalHistoryForSlot(ctx, publicKeys[i][:], proposal.Slot)
			require.NoError(t, err)
			require.DeepEqual(
				t,
				params.BeaconConfig().ZeroHash[:],
				receivedProposalSigningRoot,
				"Imported proposal signing root is different than the empty default",
			)
		}
	}
}

func TestStore_ImportInterchangeData_OK(t *testing.T) {
	ctx := context.Background()
	numValidators := 10
	publicKeys := spTest.CreateRandomPubKeys(t, numValidators)
	validatorDB := dbtest.SetupDB(t, publicKeys)

	// First we setup some mock attesting and proposal histories and create a mock
	// standard slashing protection format JSON struct.
	attestingHistory, proposalHistory := spTest.MockAttestingAndProposalHistories(t, numValidators)
	standardProtectionFormat := spTest.MockSlashingProtectionJSON(t, publicKeys, attestingHistory, proposalHistory)

	// We encode the standard slashing protection struct into a JSON format.
	blob, err := json.Marshal(standardProtectionFormat)
	require.NoError(t, err)
	buf := bytes.NewBuffer(blob)

	// Next, we attempt to import it into our validator database.
	err = interchangeformat.ImportStandardProtectionJSON(ctx, validatorDB, buf)
	require.NoError(t, err)

	// Next, we attempt to retrieve the attesting and proposals histories from our database and
	// verify those indeed match the originally generated mock histories.
	receivedAttestingHistory, err := validatorDB.AttestationHistoryForPubKeysV2(ctx, publicKeys)
	require.NoError(t, err)
	for i := 0; i < len(publicKeys); i++ {
		require.DeepEqual(
			t,
			attestingHistory[i],
			receivedAttestingHistory[publicKeys[i]],
			"We should have stored any attesting history",
		)
		proposals := proposalHistory[i].Proposals
		for _, proposal := range proposals {
			receivedProposalSigningRoot, err := validatorDB.ProposalHistoryForSlot(ctx, publicKeys[i][:], proposal.Slot)
			require.NoError(t, err)
			require.DeepEqual(
				t,
				receivedProposalSigningRoot,
				proposal.SigningRoot,
				"Imported proposals are different then the generated ones",
			)
		}
	}
}

func Test_validateMetadata(t *testing.T) {
	tests := []struct {
		name            string
		interchangeJSON *interchangeformat.EIPSlashingProtectionFormat
		wantErr         bool
	}{
		{
			name: "Incorrect version for EIP format should fail",
			interchangeJSON: &interchangeformat.EIPSlashingProtectionFormat{
				Metadata: struct {
					InterchangeFormatVersion string `json:"interchange_format_version"`
					GenesisValidatorsRoot    string `json:"genesis_validators_root"`
				}{
					InterchangeFormatVersion: "1",
				},
			},
			wantErr: true,
		},
		{
			name: "Junk data for version should fail",
			interchangeJSON: &interchangeformat.EIPSlashingProtectionFormat{
				Metadata: struct {
					InterchangeFormatVersion string `json:"interchange_format_version"`
					GenesisValidatorsRoot    string `json:"genesis_validators_root"`
				}{
					InterchangeFormatVersion: "asdljas$d",
				},
			},
			wantErr: true,
		},
		{
			name: "Proper version field should pass",
			interchangeJSON: &interchangeformat.EIPSlashingProtectionFormat{
				Metadata: struct {
					InterchangeFormatVersion string `json:"interchange_format_version"`
					GenesisValidatorsRoot    string `json:"genesis_validators_root"`
				}{
					InterchangeFormatVersion: interchangeformat.INTERCHANGE_FORMAT_VERSION,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validatorDB := dbtest.SetupDB(t, nil)
			ctx := context.Background()
			if err := interchangeformat.ValidateMetadata(ctx, validatorDB, tt.interchangeJSON); (err != nil) != tt.wantErr {
				t.Errorf("ValidateMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_parseUniqueSignedBlocksByPubKey(t *testing.T) {
	numValidators := 4
	pubKeys := spTest.CreateRandomPubKeys(t, numValidators)
	roots := spTest.CreateRandomRoots(t, numValidators)
	tests := []struct {
		name    string
		data    []*interchangeformat.ProtectionData
		want    map[[48]byte][]*interchangeformat.SignedBlock
		wantErr bool
	}{
		{
			name: "nil values are skipped",
			data: []*interchangeformat.ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*interchangeformat.SignedBlock{
						{
							Slot:        "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
						nil,
					},
				},
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*interchangeformat.SignedBlock{
						{
							Slot:        "3",
							SigningRoot: fmt.Sprintf("%x", roots[2]),
						},
					},
				},
			},
			want: map[[48]byte][]*interchangeformat.SignedBlock{
				pubKeys[0]: {
					{
						Slot:        "1",
						SigningRoot: fmt.Sprintf("%x", roots[0]),
					},
					{
						Slot:        "3",
						SigningRoot: fmt.Sprintf("%x", roots[2]),
					},
				},
			},
		},
		{
			name: "same blocks but different public keys are parsed correctly",
			data: []*interchangeformat.ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*interchangeformat.SignedBlock{
						{
							Slot:        "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
						{
							Slot:        "2",
							SigningRoot: fmt.Sprintf("%x", roots[1]),
						},
					},
				},
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[1]),
					SignedBlocks: []*interchangeformat.SignedBlock{
						{
							Slot:        "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
						{
							Slot:        "2",
							SigningRoot: fmt.Sprintf("%x", roots[1]),
						},
					},
				},
			},
			want: map[[48]byte][]*interchangeformat.SignedBlock{
				pubKeys[0]: {
					{
						Slot:        "1",
						SigningRoot: fmt.Sprintf("%x", roots[0]),
					},
					{
						Slot:        "2",
						SigningRoot: fmt.Sprintf("%x", roots[1]),
					},
				},
				pubKeys[1]: {
					{
						Slot:        "1",
						SigningRoot: fmt.Sprintf("%x", roots[0]),
					},
					{
						Slot:        "2",
						SigningRoot: fmt.Sprintf("%x", roots[1]),
					},
				},
			},
		},
		{
			name: "disjoint sets of signed blocks by the same public key are parsed correctly",
			data: []*interchangeformat.ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*interchangeformat.SignedBlock{
						{
							Slot:        "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
						{
							Slot:        "2",
							SigningRoot: fmt.Sprintf("%x", roots[1]),
						},
					},
				},
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*interchangeformat.SignedBlock{
						{
							Slot:        "3",
							SigningRoot: fmt.Sprintf("%x", roots[2]),
						},
					},
				},
			},
			want: map[[48]byte][]*interchangeformat.SignedBlock{
				pubKeys[0]: {
					{
						Slot:        "1",
						SigningRoot: fmt.Sprintf("%x", roots[0]),
					},
					{
						Slot:        "2",
						SigningRoot: fmt.Sprintf("%x", roots[1]),
					},
					{
						Slot:        "3",
						SigningRoot: fmt.Sprintf("%x", roots[2]),
					},
				},
			},
		},
		{
			name: "full duplicate entries are uniquely parsed",
			data: []*interchangeformat.ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*interchangeformat.SignedBlock{
						{
							Slot:        "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
					},
				},
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*interchangeformat.SignedBlock{
						{
							Slot:        "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
					},
				},
			},
			want: map[[48]byte][]*interchangeformat.SignedBlock{
				pubKeys[0]: {
					{
						Slot:        "1",
						SigningRoot: fmt.Sprintf("%x", roots[0]),
					},
				},
			},
		},
		{
			name: "intersecting duplicate public key entries are handled properly",
			data: []*interchangeformat.ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*interchangeformat.SignedBlock{
						{
							Slot:        "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
						{
							Slot:        "2",
							SigningRoot: fmt.Sprintf("%x", roots[1]),
						},
					},
				},
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*interchangeformat.SignedBlock{
						{
							Slot:        "2",
							SigningRoot: fmt.Sprintf("%x", roots[1]),
						},
						{
							Slot:        "3",
							SigningRoot: fmt.Sprintf("%x", roots[2]),
						},
					},
				},
			},
			want: map[[48]byte][]*interchangeformat.SignedBlock{
				pubKeys[0]: {
					{
						Slot:        "1",
						SigningRoot: fmt.Sprintf("%x", roots[0]),
					},
					{
						Slot:        "2",
						SigningRoot: fmt.Sprintf("%x", roots[1]),
					},
					{
						Slot:        "3",
						SigningRoot: fmt.Sprintf("%x", roots[2]),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := interchangeformat.ParseUniqueSignedBlocksByPubKey(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseUniqueSignedBlocksByPubKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseUniqueSignedBlocksByPubKey() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseUniqueSignedAttestationsByPubKey(t *testing.T) {
	numValidators := 4
	pubKeys := spTest.CreateRandomPubKeys(t, numValidators)
	roots := spTest.CreateRandomRoots(t, numValidators)
	tests := []struct {
		name    string
		data    []*interchangeformat.ProtectionData
		want    map[[48]byte][]*interchangeformat.SignedAttestation
		wantErr bool
	}{
		{
			name: "nil values are skipped",
			data: []*interchangeformat.ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*interchangeformat.SignedAttestation{
						{
							SourceEpoch: "1",
							TargetEpoch: "3",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
						nil,
					},
				},
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*interchangeformat.SignedAttestation{
						{
							SourceEpoch: "3",
							TargetEpoch: "5",
							SigningRoot: fmt.Sprintf("%x", roots[2]),
						},
					},
				},
			},
			want: map[[48]byte][]*interchangeformat.SignedAttestation{
				pubKeys[0]: {
					{
						SourceEpoch: "1",
						TargetEpoch: "3",
						SigningRoot: fmt.Sprintf("%x", roots[0]),
					},
					{
						SourceEpoch: "3",
						TargetEpoch: "5",
						SigningRoot: fmt.Sprintf("%x", roots[2]),
					},
				},
			},
		},
		{
			name: "same blocks but different public keys are parsed correctly",
			data: []*interchangeformat.ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*interchangeformat.SignedAttestation{
						{
							SourceEpoch: "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
						{
							SourceEpoch: "2",
							SigningRoot: fmt.Sprintf("%x", roots[1]),
						},
					},
				},
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[1]),
					SignedAttestations: []*interchangeformat.SignedAttestation{
						{
							SourceEpoch: "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
						{
							SourceEpoch: "2",
							SigningRoot: fmt.Sprintf("%x", roots[1]),
						},
					},
				},
			},
			want: map[[48]byte][]*interchangeformat.SignedAttestation{
				pubKeys[0]: {
					{
						SourceEpoch: "1",
						SigningRoot: fmt.Sprintf("%x", roots[0]),
					},
					{
						SourceEpoch: "2",
						SigningRoot: fmt.Sprintf("%x", roots[1]),
					},
				},
				pubKeys[1]: {
					{
						SourceEpoch: "1",
						SigningRoot: fmt.Sprintf("%x", roots[0]),
					},
					{
						SourceEpoch: "2",
						SigningRoot: fmt.Sprintf("%x", roots[1]),
					},
				},
			},
		},
		{
			name: "disjoint sets of signed blocks by the same public key are parsed correctly",
			data: []*interchangeformat.ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*interchangeformat.SignedAttestation{
						{
							SourceEpoch: "1",
							TargetEpoch: "3",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
						{
							SourceEpoch: "2",
							TargetEpoch: "4",
							SigningRoot: fmt.Sprintf("%x", roots[1]),
						},
					},
				},
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*interchangeformat.SignedAttestation{
						{
							SourceEpoch: "3",
							TargetEpoch: "5",
							SigningRoot: fmt.Sprintf("%x", roots[2]),
						},
					},
				},
			},
			want: map[[48]byte][]*interchangeformat.SignedAttestation{
				pubKeys[0]: {
					{
						SourceEpoch: "1",
						TargetEpoch: "3",
						SigningRoot: fmt.Sprintf("%x", roots[0]),
					},
					{
						SourceEpoch: "2",
						TargetEpoch: "4",
						SigningRoot: fmt.Sprintf("%x", roots[1]),
					},
					{
						SourceEpoch: "3",
						TargetEpoch: "5",
						SigningRoot: fmt.Sprintf("%x", roots[2]),
					},
				},
			},
		},
		{
			name: "full duplicate entries are uniquely parsed",
			data: []*interchangeformat.ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*interchangeformat.SignedAttestation{
						{
							SourceEpoch: "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
					},
				},
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*interchangeformat.SignedAttestation{
						{
							SourceEpoch: "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
					},
				},
			},
			want: map[[48]byte][]*interchangeformat.SignedAttestation{
				pubKeys[0]: {
					{
						SourceEpoch: "1",
						SigningRoot: fmt.Sprintf("%x", roots[0]),
					},
				},
			},
		},
		{
			name: "intersecting duplicate public key entries are handled properly",
			data: []*interchangeformat.ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*interchangeformat.SignedAttestation{
						{
							SourceEpoch: "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
						{
							SourceEpoch: "2",
							SigningRoot: fmt.Sprintf("%x", roots[1]),
						},
					},
				},
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*interchangeformat.SignedAttestation{
						{
							SourceEpoch: "2",
							SigningRoot: fmt.Sprintf("%x", roots[1]),
						},
						{
							SourceEpoch: "3",
							SigningRoot: fmt.Sprintf("%x", roots[2]),
						},
					},
				},
			},
			want: map[[48]byte][]*interchangeformat.SignedAttestation{
				pubKeys[0]: {
					{
						SourceEpoch: "1",
						SigningRoot: fmt.Sprintf("%x", roots[0]),
					},
					{
						SourceEpoch: "2",
						SigningRoot: fmt.Sprintf("%x", roots[1]),
					},
					{
						SourceEpoch: "3",
						SigningRoot: fmt.Sprintf("%x", roots[2]),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := interchangeformat.ParseUniqueSignedAttestationsByPubKey(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseUniqueSignedAttestationsByPubKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseUniqueSignedAttestationsByPubKey() got = %v, want %v", got, tt.want)
			}
		})
	}
}

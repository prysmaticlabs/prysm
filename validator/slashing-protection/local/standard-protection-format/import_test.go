package interchangeformat_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
	interchangeformat "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/standard-protection-format"
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
			receivedProposalSigningRoot, _, err := validatorDB.ProposalHistoryForSlot(ctx, publicKeys[i], proposal.Slot)
			require.NoError(t, err)
			require.DeepEqual(
				t,
				params.BeaconConfig().ZeroHash,
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
			receivedProposalSigningRoot, _, err := validatorDB.ProposalHistoryForSlot(ctx, publicKeys[i], proposal.Slot)
			require.NoError(t, err)
			require.DeepEqual(
				t,
				receivedProposalSigningRoot[:],
				proposal.SigningRoot,
				"Imported proposals are different then the generated ones",
			)
		}
	}
}

func Test_validateMetadata(t *testing.T) {
	goodRoot := [32]byte{1}
	goodStr := make([]byte, hex.EncodedLen(len(goodRoot)))
	hex.Encode(goodStr, goodRoot[:])
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
					GenesisValidatorsRoot:    string(goodStr),
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
					GenesisValidatorsRoot:    string(goodStr),
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
					GenesisValidatorsRoot:    string(goodStr),
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

func Test_validateMetadataGenesisValidatorRoot(t *testing.T) {
	goodRoot := [32]byte{1}
	goodStr := make([]byte, hex.EncodedLen(len(goodRoot)))
	hex.Encode(goodStr, goodRoot[:])
	secondRoot := [32]byte{2}
	secondStr := make([]byte, hex.EncodedLen(len(secondRoot)))
	hex.Encode(secondStr, secondRoot[:])

	tests := []struct {
		name                   string
		interchangeJSON        *interchangeformat.EIPSlashingProtectionFormat
		dbGenesisValidatorRoot []byte
		wantErr                bool
	}{
		{
			name: "Same genesis roots should not fail",
			interchangeJSON: &interchangeformat.EIPSlashingProtectionFormat{
				Metadata: struct {
					InterchangeFormatVersion string `json:"interchange_format_version"`
					GenesisValidatorsRoot    string `json:"genesis_validators_root"`
				}{
					InterchangeFormatVersion: interchangeformat.INTERCHANGE_FORMAT_VERSION,
					GenesisValidatorsRoot:    string(goodStr),
				},
			},
			dbGenesisValidatorRoot: goodRoot[:],
			wantErr:                false,
		},
		{
			name: "Different genesis roots should not fail",
			interchangeJSON: &interchangeformat.EIPSlashingProtectionFormat{
				Metadata: struct {
					InterchangeFormatVersion string `json:"interchange_format_version"`
					GenesisValidatorsRoot    string `json:"genesis_validators_root"`
				}{
					InterchangeFormatVersion: interchangeformat.INTERCHANGE_FORMAT_VERSION,
					GenesisValidatorsRoot:    string(secondStr),
				},
			},
			dbGenesisValidatorRoot: goodRoot[:],
			wantErr:                true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validatorDB := dbtest.SetupDB(t, nil)
			ctx := context.Background()
			require.NoError(t, validatorDB.SaveGenesisValidatorsRoot(ctx, tt.dbGenesisValidatorRoot))
			err := interchangeformat.ValidateMetadata(ctx, validatorDB, tt.interchangeJSON)
			if tt.wantErr {
				require.ErrorContains(t, "genesis validator root doesnt match the one that is stored", err)
			} else {
				require.NoError(t, err)
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

func Test_saveLowestSourceTargetToDBt_Ok(t *testing.T) {
	ctx := context.Background()
	numValidators := 2
	publicKeys := spTest.CreateRandomPubKeys(t, numValidators)
	validatorDB := dbtest.SetupDB(t, publicKeys)

	m := make(map[[48]byte][]*interchangeformat.SignedAttestation)
	m[publicKeys[0]] = []*interchangeformat.SignedAttestation{{SourceEpoch: "1", TargetEpoch: "2"}, {SourceEpoch: "3", TargetEpoch: "4"}}
	m[publicKeys[1]] = []*interchangeformat.SignedAttestation{{SourceEpoch: "8", TargetEpoch: "7"}, {SourceEpoch: "6", TargetEpoch: "5"}}
	require.NoError(t, interchangeformat.SaveLowestSourceTargetToDB(ctx, validatorDB, m))

	got, e, err := validatorDB.LowestSignedTargetEpoch(ctx, publicKeys[0])
	require.NoError(t, err)
	require.Equal(t, true, e)
	require.Equal(t, uint64(2), got)
	got, e, err = validatorDB.LowestSignedTargetEpoch(ctx, publicKeys[1])
	require.NoError(t, err)
	require.Equal(t, true, e)
	require.Equal(t, uint64(5), got)
	got, e, err = validatorDB.LowestSignedSourceEpoch(ctx, publicKeys[0])
	require.NoError(t, err)
	require.Equal(t, true, e)
	require.Equal(t, uint64(1), got)
	got, e, err = validatorDB.LowestSignedSourceEpoch(ctx, publicKeys[1])
	require.NoError(t, err)
	require.Equal(t, true, e)
	require.Equal(t, uint64(6), got)
}

func Test_filterSlashablePubKeysFromBlocks(t *testing.T) {
	var tests = []struct {
		name     string
		expected [][48]byte
		given    map[[48]byte][]*interchangeformat.SignedBlock
	}{
		{
			name:     "No slashable keys returns empty",
			expected: make([][48]byte, 0),
			given: map[[48]byte][]*interchangeformat.SignedBlock{
				{1}: {
					{
						Slot: "1",
					},
					{
						Slot: "2",
					},
				},
				{2}: {
					{
						Slot: "2",
					},
					{
						Slot: "3",
					},
				},
			},
		},
		{
			name:     "Empty data returns empty",
			expected: make([][48]byte, 0),
			given:    make(map[[48]byte][]*interchangeformat.SignedBlock),
		},
		{
			name: "Properly finds public keys with slashable data",
			expected: [][48]byte{
				{1}, {3},
			},
			given: map[[48]byte][]*interchangeformat.SignedBlock{
				{1}: {
					{
						Slot: "1",
					},
					{
						Slot: "1",
					},
					{
						Slot: "2",
					},
				},
				{2}: {
					{
						Slot: "2",
					},
					{
						Slot: "3",
					},
				},
				{3}: {
					{
						Slot: "3",
					},
					{
						Slot: "3",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			slashablePubKeys := filterSlashablePubKeysFromBlocks(context.Background(), tt.given)
			require.DeepEqual(t, tt.expected, slashablePubKeys)
		})
	}
}

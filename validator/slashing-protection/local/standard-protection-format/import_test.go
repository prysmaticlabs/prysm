package interchangeformat

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestStore_ImportInterchangeData_BadJSON(t *testing.T) {
	ctx := context.Background()
	validatorDB := dbtest.SetupDB(t, nil)

	buf := bytes.NewBuffer([]byte("helloworld"))
	err := ImportStandardProtectionJSON(ctx, validatorDB, buf)
	require.ErrorContains(t, "could not unmarshal slashing protection JSON file", err)
}

func TestStore_ImportInterchangeData_NilData_FailsSilently(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	validatorDB := dbtest.SetupDB(t, nil)

	interchangeJSON := &EIPSlashingProtectionFormat{}
	encoded, err := json.Marshal(interchangeJSON)
	require.NoError(t, err)

	buf := bytes.NewBuffer(encoded)
	err = ImportStandardProtectionJSON(ctx, validatorDB, buf)
	require.NoError(t, err)
	require.LogsContain(t, hook, "No slashing protection data to import")
}

func TestStore_ImportInterchangeData_BadFormat_PreventsDBWrites(t *testing.T) {
	ctx := context.Background()
	numValidators := 10
	publicKeys := createRandomPubKeys(t, numValidators)
	validatorDB := dbtest.SetupDB(t, publicKeys)

	// First we setup some mock attesting and proposal histories and create a mock
	// standard slashing protection format JSON struct.
	attestingHistory, proposalHistory := mockAttestingAndProposalHistories(t, numValidators)
	standardProtectionFormat := mockSlashingProtectionJSON(t, publicKeys, attestingHistory, proposalHistory)

	// We replace a slot of one of the blocks with junk data.
	standardProtectionFormat.Data[0].SignedBlocks[0].Slot = "BadSlot"

	// We encode the standard slashing protection struct into a JSON format.
	blob, err := json.Marshal(standardProtectionFormat)
	require.NoError(t, err)
	buf := bytes.NewBuffer(blob)

	// Next, we attempt to import it into our validator database and check that
	// we obtain an error during the import process.
	err = ImportStandardProtectionJSON(ctx, validatorDB, buf)
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
	publicKeys := createRandomPubKeys(t, numValidators)
	validatorDB := dbtest.SetupDB(t, publicKeys)

	// First we setup some mock attesting and proposal histories and create a mock
	// standard slashing protection format JSON struct.
	attestingHistory, proposalHistory := mockAttestingAndProposalHistories(t, numValidators)
	standardProtectionFormat := mockSlashingProtectionJSON(t, publicKeys, attestingHistory, proposalHistory)

	// We encode the standard slashing protection struct into a JSON format.
	blob, err := json.Marshal(standardProtectionFormat)
	require.NoError(t, err)
	buf := bytes.NewBuffer(blob)

	// Next, we attempt to import it into our validator database.
	err = ImportStandardProtectionJSON(ctx, validatorDB, buf)
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
		name                   string
		interchangeJSON        *EIPSlashingProtectionFormat
		dbGenesisValidatorRoot []byte
		wantErr                bool
		wantFatal              string
	}{
		{
			name: "Incorrect version for EIP format should fail",
			interchangeJSON: &EIPSlashingProtectionFormat{
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
			interchangeJSON: &EIPSlashingProtectionFormat{
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
			interchangeJSON: &EIPSlashingProtectionFormat{
				Metadata: struct {
					InterchangeFormatVersion string `json:"interchange_format_version"`
					GenesisValidatorsRoot    string `json:"genesis_validators_root"`
				}{
					InterchangeFormatVersion: INTERCHANGE_FORMAT_VERSION,
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
			if err := validateMetadata(ctx, validatorDB, tt.interchangeJSON); (err != nil) != tt.wantErr {
				t.Errorf("validateMetadata() error = %v, wantErr %v", err, tt.wantErr)
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
		interchangeJSON        *EIPSlashingProtectionFormat
		dbGenesisValidatorRoot []byte
		wantErr                bool
	}{
		{
			name: "Same genesis roots should not fail",
			interchangeJSON: &EIPSlashingProtectionFormat{
				Metadata: struct {
					InterchangeFormatVersion string `json:"interchange_format_version"`
					GenesisValidatorsRoot    string `json:"genesis_validators_root"`
				}{
					InterchangeFormatVersion: INTERCHANGE_FORMAT_VERSION,
					GenesisValidatorsRoot:    string(goodStr),
				},
			},
			dbGenesisValidatorRoot: goodRoot[:],
			wantErr:                false,
		},
		{
			name: "Different genesis roots should not fail",
			interchangeJSON: &EIPSlashingProtectionFormat{
				Metadata: struct {
					InterchangeFormatVersion string `json:"interchange_format_version"`
					GenesisValidatorsRoot    string `json:"genesis_validators_root"`
				}{
					InterchangeFormatVersion: INTERCHANGE_FORMAT_VERSION,
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
			err := validateMetadata(ctx, validatorDB, tt.interchangeJSON)
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
	pubKeys := createRandomPubKeys(t, numValidators)
	roots := createRandomRoots(t, numValidators)
	tests := []struct {
		name    string
		data    []*ProtectionData
		want    map[[48]byte][]*SignedBlock
		wantErr bool
	}{
		{
			name: "nil values are skipped",
			data: []*ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*SignedBlock{
						{
							Slot:        "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
						nil,
					},
				},
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*SignedBlock{
						{
							Slot:        "3",
							SigningRoot: fmt.Sprintf("%x", roots[2]),
						},
					},
				},
			},
			want: map[[48]byte][]*SignedBlock{
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
			data: []*ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*SignedBlock{
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
					SignedBlocks: []*SignedBlock{
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
			want: map[[48]byte][]*SignedBlock{
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
			data: []*ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*SignedBlock{
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
					SignedBlocks: []*SignedBlock{
						{
							Slot:        "3",
							SigningRoot: fmt.Sprintf("%x", roots[2]),
						},
					},
				},
			},
			want: map[[48]byte][]*SignedBlock{
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
			data: []*ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*SignedBlock{
						{
							Slot:        "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
					},
				},
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*SignedBlock{
						{
							Slot:        "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
					},
				},
			},
			want: map[[48]byte][]*SignedBlock{
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
			data: []*ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedBlocks: []*SignedBlock{
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
					SignedBlocks: []*SignedBlock{
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
			want: map[[48]byte][]*SignedBlock{
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
			got, err := parseUniqueSignedBlocksByPubKey(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseUniqueSignedBlocksByPubKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseUniqueSignedBlocksByPubKey() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseUniqueSignedAttestationsByPubKey(t *testing.T) {
	numValidators := 4
	pubKeys := createRandomPubKeys(t, numValidators)
	roots := createRandomRoots(t, numValidators)
	tests := []struct {
		name    string
		data    []*ProtectionData
		want    map[[48]byte][]*SignedAttestation
		wantErr bool
	}{
		{
			name: "nil values are skipped",
			data: []*ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*SignedAttestation{
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
					SignedAttestations: []*SignedAttestation{
						{
							SourceEpoch: "3",
							TargetEpoch: "5",
							SigningRoot: fmt.Sprintf("%x", roots[2]),
						},
					},
				},
			},
			want: map[[48]byte][]*SignedAttestation{
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
			data: []*ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*SignedAttestation{
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
					SignedAttestations: []*SignedAttestation{
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
			want: map[[48]byte][]*SignedAttestation{
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
			data: []*ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*SignedAttestation{
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
					SignedAttestations: []*SignedAttestation{
						{
							SourceEpoch: "3",
							TargetEpoch: "5",
							SigningRoot: fmt.Sprintf("%x", roots[2]),
						},
					},
				},
			},
			want: map[[48]byte][]*SignedAttestation{
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
			data: []*ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*SignedAttestation{
						{
							SourceEpoch: "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
					},
				},
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*SignedAttestation{
						{
							SourceEpoch: "1",
							SigningRoot: fmt.Sprintf("%x", roots[0]),
						},
					},
				},
			},
			want: map[[48]byte][]*SignedAttestation{
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
			data: []*ProtectionData{
				{
					Pubkey: fmt.Sprintf("%x", pubKeys[0]),
					SignedAttestations: []*SignedAttestation{
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
					SignedAttestations: []*SignedAttestation{
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
			want: map[[48]byte][]*SignedAttestation{
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
			got, err := parseUniqueSignedAttestationsByPubKey(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseUniqueSignedAttestationsByPubKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseUniqueSignedAttestationsByPubKey() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_saveLowestSourceTargetToDBt_Ok(t *testing.T) {
	ctx := context.Background()
	numValidators := 2
	publicKeys := createRandomPubKeys(t, numValidators)
	validatorDB := dbtest.SetupDB(t, publicKeys)

	m := make(map[[48]byte][]*SignedAttestation)
	m[publicKeys[0]] = []*SignedAttestation{{SourceEpoch: "1", TargetEpoch: "2"}, {SourceEpoch: "3", TargetEpoch: "4"}}
	m[publicKeys[1]] = []*SignedAttestation{{SourceEpoch: "8", TargetEpoch: "7"}, {SourceEpoch: "6", TargetEpoch: "5"}}
	require.NoError(t, saveLowestSourceTargetToDB(ctx, validatorDB, m))

	got, err := validatorDB.LowestSignedTargetEpoch(ctx, publicKeys[0])
	require.NoError(t, err)
	require.Equal(t, uint64(2), got)
	got, err = validatorDB.LowestSignedTargetEpoch(ctx, publicKeys[1])
	require.NoError(t, err)
	require.Equal(t, uint64(5), got)
	got, err = validatorDB.LowestSignedSourceEpoch(ctx, publicKeys[0])
	require.NoError(t, err)
	require.Equal(t, uint64(1), got)
	got, err = validatorDB.LowestSignedSourceEpoch(ctx, publicKeys[1])
	require.NoError(t, err)
	require.Equal(t, uint64(6), got)
}

func mockSlashingProtectionJSON(
	t *testing.T,
	publicKeys [][48]byte,
	attestingHistories []kv.EncHistoryData,
	proposalHistories []kv.ProposalHistoryForPubkey,
) *EIPSlashingProtectionFormat {
	standardProtectionFormat := &EIPSlashingProtectionFormat{}
	standardProtectionFormat.Metadata.GenesisValidatorsRoot = fmt.Sprintf("%#x", bytesutil.PadTo([]byte{32}, 32))
	standardProtectionFormat.Metadata.InterchangeFormatVersion = INTERCHANGE_FORMAT_VERSION
	ctx := context.Background()
	for i := 0; i < len(publicKeys); i++ {
		data := &ProtectionData{
			Pubkey: fmt.Sprintf("%#x", publicKeys[i]),
		}
		highestEpochWritten, err := attestingHistories[i].GetLatestEpochWritten(ctx)
		require.NoError(t, err)
		for target := uint64(0); target <= highestEpochWritten; target++ {
			hd, err := attestingHistories[i].GetTargetData(ctx, target)
			require.NoError(t, err)
			data.SignedAttestations = append(data.SignedAttestations, &SignedAttestation{
				TargetEpoch: fmt.Sprintf("%d", target),
				SourceEpoch: fmt.Sprintf("%d", hd.Source),
				SigningRoot: fmt.Sprintf("%#x", hd.SigningRoot),
			})
		}
		for target := uint64(0); target < highestEpochWritten; target++ {
			proposal := proposalHistories[i].Proposals[target]
			block := &SignedBlock{
				Slot:        fmt.Sprintf("%d", proposal.Slot),
				SigningRoot: fmt.Sprintf("%#x", proposal.SigningRoot),
			}
			data.SignedBlocks = append(data.SignedBlocks, block)

		}
		standardProtectionFormat.Data = append(standardProtectionFormat.Data, data)
	}
	return standardProtectionFormat
}

func mockAttestingAndProposalHistories(t *testing.T, numValidators int) ([]kv.EncHistoryData, []kv.ProposalHistoryForPubkey) {
	// deduplicate and transform them into our internal format.
	attData := make([]kv.EncHistoryData, numValidators)
	proposalData := make([]kv.ProposalHistoryForPubkey, numValidators)
	gen := rand.NewGenerator()
	ctx := context.Background()
	for v := 0; v < numValidators; v++ {
		var err error
		latestTarget := gen.Intn(int(params.BeaconConfig().WeakSubjectivityPeriod) / 1000)
		hd := kv.NewAttestationHistoryArray(uint64(latestTarget))
		proposals := make([]kv.Proposal, 0)
		for i := 1; i < latestTarget; i++ {
			signingRoot := [32]byte{}
			signingRootStr := fmt.Sprintf("%d", i)
			copy(signingRoot[:], signingRootStr)
			historyData := &kv.HistoryData{
				Source:      uint64(gen.Intn(100000)),
				SigningRoot: signingRoot[:],
			}
			hd, err = hd.SetTargetData(ctx, uint64(i), historyData)
			require.NoError(t, err)
		}
		for i := 1; i <= latestTarget; i++ {
			signingRoot := [32]byte{}
			signingRootStr := fmt.Sprintf("%d", i)
			copy(signingRoot[:], signingRootStr)
			proposals = append(proposals, kv.Proposal{
				Slot:        uint64(i),
				SigningRoot: signingRoot[:],
			})
		}
		proposalData[v] = kv.ProposalHistoryForPubkey{Proposals: proposals}
		hd, err = hd.SetLatestEpochWritten(ctx, uint64(latestTarget))
		require.NoError(t, err)
		attData[v] = hd
	}
	return attData, proposalData
}

func createRandomPubKeys(t *testing.T, numValidators int) [][48]byte {
	pubKeys := make([][48]byte, numValidators)
	for i := 0; i < numValidators; i++ {
		randKey, err := bls.RandKey()
		require.NoError(t, err)
		copy(pubKeys[i][:], randKey.PublicKey().Marshal())
	}
	return pubKeys
}

func createRandomRoots(t *testing.T, numRoots int) [][32]byte {
	roots := make([][32]byte, numRoots)
	for i := 0; i < numRoots; i++ {
		roots[i] = hashutil.Hash([]byte(fmt.Sprintf("%d", i)))
	}
	return roots
}

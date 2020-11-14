package interchangeformat

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
)

const numValidators = 10

func TestStore_ImportInterchangeData_BadJSON(t *testing.T) {
	ctx := context.Background()
	publicKeys := createRandomPubKeys(t)
	validatorDB := dbtest.SetupDB(t, publicKeys)

	buf := bytes.NewBuffer([]byte("helloworld"))
	err := ImportStandardProtectionJSON(ctx, validatorDB, buf)
	require.ErrorContains(t, "could not unmarshal slashing protection JSON file", err)
}

func TestStore_ImportInterchangeData_BadFormat_PreventsDBWrites(t *testing.T) {
	ctx := context.Background()
	publicKeys := createRandomPubKeys(t)
	validatorDB := dbtest.SetupDB(t, publicKeys)

	// First we setup some mock attesting and proposal histories and create a mock
	// standard slashing protection format JSON struct.
	attestingHistory, proposalHistory := mockAttestingAndProposalHistories(t)
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
	// verify those indeed match the originally generated mock histories.
	receivedAttestingHistory, err := validatorDB.AttestationHistoryForPubKeysV2(ctx, publicKeys)
	require.NoError(t, err)
	for i := 0; i < numValidators; i++ {
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
	publicKeys := createRandomPubKeys(t)
	validatorDB := dbtest.SetupDB(t, publicKeys)

	// First we setup some mock attesting and proposal histories and create a mock
	// standard slashing protection format JSON struct.
	attestingHistory, proposalHistory := mockAttestingAndProposalHistories(t)
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
	for i := 0; i < numValidators; i++ {
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

func mockSlashingProtectionJSON(
	t *testing.T,
	publicKeys [][48]byte,
	attestingHistories []kv.EncHistoryData,
	proposalHistories []kv.ProposalHistoryForPubkey,
) *EIPSlashingProtectionFormat {
	standardProtectionFormat := &EIPSlashingProtectionFormat{}
	standardProtectionFormat.Metadata.GenesisValidatorsRoot = hex.EncodeToString(bytesutil.PadTo([]byte{32}, 32))
	standardProtectionFormat.Metadata.InterchangeFormatVersion = "5"
	ctx := context.Background()
	for i := 0; i < numValidators; i++ {
		data := &ProtectionData{
			Pubkey: hex.EncodeToString(publicKeys[i][:]),
		}
		highestEpochWritten, err := attestingHistories[i].GetLatestEpochWritten(ctx)
		require.NoError(t, err)
		for target := uint64(0); target <= highestEpochWritten; target++ {
			hd, err := attestingHistories[i].GetTargetData(ctx, target)
			require.NoError(t, err)
			data.SignedAttestations = append(data.SignedAttestations, &SignedAttestation{
				TargetEpoch: strconv.FormatUint(target, 10),
				SourceEpoch: strconv.FormatUint(hd.Source, 10),
				SigningRoot: hex.EncodeToString(hd.SigningRoot),
			})
		}
		for target := uint64(0); target < highestEpochWritten; target++ {
			proposal := proposalHistories[i].Proposals[target]
			block := &SignedBlock{
				Slot:        strconv.FormatUint(proposal.Slot, 10),
				SigningRoot: hex.EncodeToString(proposal.SigningRoot),
			}
			data.SignedBlocks = append(data.SignedBlocks, block)

		}
		standardProtectionFormat.Data = append(standardProtectionFormat.Data, data)
	}
	return standardProtectionFormat
}

func mockAttestingAndProposalHistories(t *testing.T) ([]kv.EncHistoryData, []kv.ProposalHistoryForPubkey) {
	attData := make([]kv.EncHistoryData, numValidators)
	proposalData := make([]kv.ProposalHistoryForPubkey, numValidators)
	gen := rand.NewGenerator()
	ctx := context.Background()
	for v := 0; v < numValidators; v++ {
		var err error
		latestTarget := gen.Intn(int(params.BeaconConfig().WeakSubjectivityPeriod) / 100)
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
		attData[v] = hd
	}
	return attData, proposalData
}

func createRandomPubKeys(t *testing.T) [][48]byte {
	pubKeys := make([][48]byte, numValidators)
	for i := 0; i < numValidators; i++ {
		randKey, err := bls.RandKey()
		require.NoError(t, err)
		copy(pubKeys[i][:], randKey.PublicKey().Marshal())
	}
	return pubKeys
}

func Test_uint64FromString(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		want    uint64
		wantErr bool
	}{
		{
			name:    "Overflow uint64 fails",
			str:     "2934890283904829038490283904829038490",
			wantErr: true,
		},
		{
			name:    "Negative number fails",
			str:     "-3",
			wantErr: true,
		},
		{
			name:    "Junk fails",
			str:     "alksjdkjasd",
			wantErr: true,
		},
		{
			name: "0 works",
			str:  "0",
			want: 0,
		},
		{
			name: "Normal uint64 works",
			str:  "23980",
			want: 23980,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := uint64FromString(tt.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("uint64FromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("uint64FromString() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pubKeyFromHex(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		want    []byte
		wantErr bool
	}{
		{
			name:    "Empty value fails due to wrong length",
			str:     "",
			wantErr: true,
		},
		{
			name:    "Junk fails",
			str:     "alksjdkjasd",
			wantErr: true,
		},
		{
			name:    "Empty value with 0x prefix fails due to wrong length",
			str:     "0x",
			wantErr: true,
		},
		{
			name:    "Works with 0x prefix and good public key",
			str:     "0xb845089a1457f811bfc000588fbb4e713669be8ce060ea6be3c6ece09afc3794106c91ca73acda5e5457122d58723bed",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pubKeyFromHex(tt.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("pubKeyFromHex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("pubKeyFromHex() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_rootFromHex(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		want    []byte
		wantErr bool
	}{
		//{
		//	name: "Works without 0x prefix and good public key",
		//	str:  "b845089a1457f811bfc000588fbb4e713669be8ce060ea6be3c6ece09afc3794106c91ca73acda5e5457122d58723bed",
		//	want: [32]byte{},
		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rootFromHex(tt.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("rootFromHex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rootFromHex() got = %v, want %v", got, tt.want)
			}
		})
	}
}

package interchangeformat

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
)

const numValidators = 10

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
	importedAttestingHistory, err := validatorDB.AttestationHistoryForPubKeysV2(ctx, publicKeys)
	require.NoError(t, err)
	for i := 0; i < numValidators; i++ {
		require.DeepEqual(
			t,
			attestingHistory[i],
			importedAttestingHistory[publicKeys[i]],
			"Imported attestation protection history is different than the generated ones",
		)
		history := proposalHistory[i]
		for _, proposal := range history.Proposals {
			expectedSigningRoot, err := validatorDB.ProposalHistoryForSlot(ctx, publicKeys[i][:], proposal.Slot)
			require.NoError(t, err)
			require.DeepEqual(
				t,
				proposal.SigningRoot[:],
				expectedSigningRoot,
				"Imported proposals are different then the generated ones",
			)
		}
	}
}

func mockSlashingProtectionJSON(
	t *testing.T,
	publicKeys [][48]byte,
	attestingHistories []*kv.EncHistoryData,
	proposalHistories []*kv.ProposalHistoryForPubkey,
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
				SigningRoot: hex.EncodeToString(hd.SigningRoot)},
			)
			sr := proposalHistories[i].Proposals[target].SigningRoot
			data.SignedBlocks = append(data.SignedBlocks, &SignedBlock{
				Slot:        strconv.FormatUint(target, 10),
				SigningRoot: hex.EncodeToString(sr[:])},
			)

		}
		standardProtectionFormat.Data = append(standardProtectionFormat.Data, data)
	}
	return standardProtectionFormat
}

func mockAttestingAndProposalHistories(t *testing.T) ([]*kv.EncHistoryData, []*kv.ProposalHistoryForPubkey) {
	attData := make([]*kv.EncHistoryData, numValidators)
	proposalData := make([]*kv.ProposalHistoryForPubkey, numValidators)
	gen := rand.NewGenerator()
	ctx := context.Background()
	for v := 0; v < numValidators; v++ {
		var err error
		latestTarget := gen.Intn(int(params.BeaconConfig().WeakSubjectivityPeriod) / 100)
		hd := kv.NewAttestationHistoryArray(uint64(latestTarget))
		proposals := make([]kv.Proposal, 0)
		for i := 1; i < latestTarget; i += 2 {
			historyData := &kv.HistoryData{Source: uint64(gen.Intn(100000)), SigningRoot: bytesutil.PadTo([]byte{byte(i)}, 32)}
			hd, err = hd.SetTargetData(ctx, uint64(i), historyData)
			require.NoError(t, err)
			proposals = append(proposals, kv.Proposal{
				Slot:        uint64(i),
				SigningRoot: [32]byte{byte(i)},
			})
		}
		proposalData[v] = &kv.ProposalHistoryForPubkey{Proposals: proposals}
		hd, err = hd.SetLatestEpochWritten(ctx, uint64(latestTarget))
		attData[v] = &hd
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

// Testing package is for testing utils to help with testing EIP-3076 Slashing Protection.
package testing

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	spFormat "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/standard-protection-format"
)

// MockSlashingProtectionJSON generates a mock EIP-3076 compliant slashing protection JSON for testing.
func MockSlashingProtectionJSON(
	t *testing.T,
	publicKeys [][48]byte,
	attestingHistories []kv.EncHistoryData,
	proposalHistories []kv.ProposalHistoryForPubkey,
) *spFormat.EIPSlashingProtectionFormat {
	standardProtectionFormat := &spFormat.EIPSlashingProtectionFormat{}
	standardProtectionFormat.Metadata.GenesisValidatorsRoot = hex.EncodeToString(bytesutil.PadTo([]byte{32}, 32))
	standardProtectionFormat.Metadata.InterchangeFormatVersion = "5"
	ctx := context.Background()
	for i := 0; i < len(publicKeys); i++ {
		data := &spFormat.ProtectionData{
			Pubkey: hex.EncodeToString(publicKeys[i][:]),
		}
		highestEpochWritten, err := attestingHistories[i].GetLatestEpochWritten(ctx)
		require.NoError(t, err)
		for target := uint64(0); target <= highestEpochWritten; target++ {
			hd, err := attestingHistories[i].GetTargetData(ctx, target)
			require.NoError(t, err)
			data.SignedAttestations = append(data.SignedAttestations, &spFormat.SignedAttestation{
				TargetEpoch: strconv.FormatUint(target, 10),
				SourceEpoch: strconv.FormatUint(hd.Source, 10),
				SigningRoot: hex.EncodeToString(hd.SigningRoot),
			})
		}
		for target := uint64(0); target < highestEpochWritten; target++ {
			proposal := proposalHistories[i].Proposals[target]
			block := &spFormat.SignedBlock{
				Slot:        strconv.FormatUint(proposal.Slot, 10),
				SigningRoot: hex.EncodeToString(proposal.SigningRoot),
			}
			data.SignedBlocks = append(data.SignedBlocks, block)

		}
		standardProtectionFormat.Data = append(standardProtectionFormat.Data, data)
	}
	return standardProtectionFormat
}

// MockAttestingAndProposalHistories generates mock attesting and proposal history data for testing.
func MockAttestingAndProposalHistories(t *testing.T, numValidators int) ([]kv.EncHistoryData, []kv.ProposalHistoryForPubkey) {
	// deduplicate and transform them into our internal format.
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
		require.NoError(t, err)
		attData[v] = hd
	}
	return attData, proposalData
}

// CreateRandomPubKeys generates any amount of random public keys for testing.
func CreateRandomPubKeys(t *testing.T, numValidators int) [][48]byte {
	pubKeys := make([][48]byte, numValidators)
	for i := 0; i < numValidators; i++ {
		randKey, err := bls.RandKey()
		require.NoError(t, err)
		copy(pubKeys[i][:], randKey.PublicKey().Marshal())
	}
	return pubKeys
}

// CreateRandomRoots generates any amount of deterministic roots for testing.
func CreateRandomRoots(t *testing.T, numRoots int) [][32]byte {
	roots := make([][32]byte, numRoots)
	for i := 0; i < numRoots; i++ {
		roots[i] = hashutil.Hash([]byte(fmt.Sprintf("%d", i)))
	}
	return roots
}

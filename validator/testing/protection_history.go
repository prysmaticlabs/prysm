package testing

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	protectionFormat "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/standard-protection-format"
)

func MockSlashingProtectionJSON(
	publicKeys [][48]byte,
	attestingHistories []kv.EncHistoryData,
	proposalHistories []kv.ProposalHistoryForPubkey,
) (*protectionFormat.EIPSlashingProtectionFormat, error) {
	standardProtectionFormat := &protectionFormat.EIPSlashingProtectionFormat{}
	standardProtectionFormat.Metadata.GenesisValidatorsRoot = fmt.Sprintf("%#x", bytesutil.PadTo([]byte{32}, 32))
	standardProtectionFormat.Metadata.InterchangeFormatVersion = protectionFormat.INTERCHANGE_FORMAT_VERSION
	ctx := context.Background()
	for i := 0; i < len(publicKeys); i++ {
		data := &protectionFormat.ProtectionData{
			Pubkey: fmt.Sprintf("%#x", publicKeys[i]),
		}
		highestEpochWritten, err := attestingHistories[i].GetLatestEpochWritten(ctx)
		if err != nil {
			return nil, err
		}
		for target := uint64(0); target <= highestEpochWritten; target++ {
			hd, err := attestingHistories[i].GetTargetData(ctx, target)
			if err != nil {
				return nil, err
			}
			data.SignedAttestations = append(data.SignedAttestations, &protectionFormat.SignedAttestation{
				TargetEpoch: fmt.Sprintf("%d", target),
				SourceEpoch: fmt.Sprintf("%d", hd.Source),
				SigningRoot: fmt.Sprintf("%#x", hd.SigningRoot),
			})
		}
		for target := uint64(0); target < highestEpochWritten; target++ {
			proposal := proposalHistories[i].Proposals[target]
			block := &protectionFormat.SignedBlock{
				Slot:        fmt.Sprintf("%d", proposal.Slot),
				SigningRoot: fmt.Sprintf("%#x", proposal.SigningRoot),
			}
			data.SignedBlocks = append(data.SignedBlocks, block)

		}
		standardProtectionFormat.Data = append(standardProtectionFormat.Data, data)
	}
	return standardProtectionFormat, nil
}

func MockAttestingAndProposalHistories(numValidators int) ([]kv.EncHistoryData, []kv.ProposalHistoryForPubkey, error) {
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
			if err != nil {
				return nil, nil, err
			}
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
		if err != nil {
			return nil, nil, err
		}
		attData[v] = hd
	}
	return attData, proposalData, nil
}

func CreateRandomPubKeys(t *testing.T, numValidators int) ([][48]byte, error) {
	pubKeys := make([][48]byte, numValidators)
	for i := 0; i < numValidators; i++ {
		randKey, err := bls.RandKey()
		if err != nil {
			return nil, err
		}
		copy(pubKeys[i][:], randKey.PublicKey().Marshal())
	}
	return pubKeys, nil
}

func CreateRandomRoots(t *testing.T, numRoots int) [][32]byte {
	roots := make([][32]byte, numRoots)
	for i := 0; i < numRoots; i++ {
		roots[i] = hashutil.Hash([]byte(fmt.Sprintf("%d", i)))
	}
	return roots
}

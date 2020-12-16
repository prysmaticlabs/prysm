package testing

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	protectionFormat "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/standard-protection-format"
)

// MockSlashingProtectionJSON creates a mock, full slashing protection JSON struct
// using attesting and proposing histories provided.
func MockSlashingProtectionJSON(
	publicKeys [][48]byte,
	attestingHistories map[[48]byte]kv.EncHistoryData,
	proposalHistories map[[48]byte]kv.ProposalHistoryForPubkey,
) (*protectionFormat.EIPSlashingProtectionFormat, error) {
	standardProtectionFormat := &protectionFormat.EIPSlashingProtectionFormat{}
	standardProtectionFormat.Metadata.GenesisValidatorsRoot = fmt.Sprintf("%#x", bytesutil.PadTo([]byte{32}, 32))
	standardProtectionFormat.Metadata.InterchangeFormatVersion = protectionFormat.INTERCHANGE_FORMAT_VERSION
	ctx := context.Background()
	for _, pubKey := range publicKeys {
		data := &protectionFormat.ProtectionData{
			Pubkey: fmt.Sprintf("%#x", pubKey),
		}
		highestEpochWritten, err := attestingHistories[pubKey].GetLatestEpochWritten(ctx)
		if err != nil {
			return nil, err
		}
		for target := uint64(0); target <= highestEpochWritten; target++ {
			hd, err := attestingHistories[pubKey].GetTargetData(ctx, target)
			if err != nil {
				return nil, err
			}
			if hd.IsEmpty() {
				continue
			}
			data.SignedAttestations = append(data.SignedAttestations, &protectionFormat.SignedAttestation{
				TargetEpoch: fmt.Sprintf("%d", target),
				SourceEpoch: fmt.Sprintf("%d", hd.Source),
				SigningRoot: fmt.Sprintf("%#x", hd.SigningRoot),
			})
		}
		for target := uint64(0); target < highestEpochWritten; target++ {
			proposal := proposalHistories[pubKey].Proposals[target]
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

// MockAttestingAndProposalHistories given a number of validators, creates mock attesting
// and proposing histories within WEAK_SUBJECTIVITY_PERIOD bounds.
func MockAttestingAndProposalHistories(
	pubKeys [][48]byte, numberOfProposals, numberOfAttestations int,
) (map[[48]byte]kv.EncHistoryData, map[[48]byte]kv.ProposalHistoryForPubkey, error) {
	attData := make(map[[48]byte]kv.EncHistoryData, len(pubKeys))
	proposalData := make(map[[48]byte]kv.ProposalHistoryForPubkey, len(pubKeys))
	ctx := context.Background()
	for v := 0; v < len(pubKeys); v++ {
		var err error
		hd := kv.NewAttestationHistoryArray(uint64(numberOfAttestations))
		proposals := make([]kv.Proposal, 0)
		for i := 1; i <= numberOfAttestations; i++ {
			signingRoot := [32]byte{}
			signingRootStr := fmt.Sprintf("%d", i)
			copy(signingRoot[:], signingRootStr)
			historyData := &kv.HistoryData{
				Source:      uint64(i - 1),
				SigningRoot: signingRoot[:],
			}
			hd, err = hd.SetTargetData(ctx, uint64(i), historyData)
			if err != nil {
				return nil, nil, err
			}
		}
		for i := 1; i <= numberOfProposals; i++ {
			signingRoot := [32]byte{}
			signingRootStr := fmt.Sprintf("%d", i)
			copy(signingRoot[:], signingRootStr)
			proposals = append(proposals, kv.Proposal{
				Slot:        uint64(i),
				SigningRoot: signingRoot[:],
			})
		}
		proposalData[pubKeys[v]] = kv.ProposalHistoryForPubkey{Proposals: proposals}
		hd, err = hd.SetLatestEpochWritten(ctx, uint64(numberOfAttestations))
		if err != nil {
			return nil, nil, err
		}
		attData[pubKeys[v]] = hd
	}
	return attData, proposalData, nil
}

// CreateRandomPubKeys --
func CreateRandomPubKeys(numValidators int) ([][48]byte, error) {
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

// CreateRandomRoots --
func CreateRandomRoots(numRoots int) [][32]byte {
	roots := make([][32]byte, numRoots)
	for i := 0; i < numRoots; i++ {
		roots[i] = hashutil.Hash([]byte(fmt.Sprintf("%d", i)))
	}
	return roots
}

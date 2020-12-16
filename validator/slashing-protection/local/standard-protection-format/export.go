package interchangeformat

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/db"
	"sort"
	"strings"
)

// ExportStandardProtectionJSON extracts all slashing protection data from a validator database
// and packages it into an EIP-3076 compliant, standard
func ExportStandardProtectionJSON(ctx context.Context, validatorDB db.Database) (*EIPSlashingProtectionFormat, error) {
	interchangeJSON := &EIPSlashingProtectionFormat{}
	genesisValidatorsRoot, err := validatorDB.GenesisValidatorsRoot(ctx)
	if err != nil {
		return nil, err
	}
	genesisRootHex, err := rootToHexString(genesisValidatorsRoot)
	if err != nil {
		return nil, err
	}
	interchangeJSON.Metadata.GenesisValidatorsRoot = genesisRootHex
	interchangeJSON.Metadata.InterchangeFormatVersion = INTERCHANGE_FORMAT_VERSION

	// Extract the existing public keys in our database.
	proposedPublicKeys, err := validatorDB.ProposedPublicKeys(ctx)
	if err != nil {
		return nil, err
	}
	attestedPublicKeys, err := validatorDB.AttestedPublicKeys(ctx)
	if err != nil {
		return nil, err
	}
	dataByPubKey := make(map[[48]byte]*ProtectionData)

	// Extract the signed proposals by public key.
	for _, pubKey := range proposedPublicKeys {
		pubKeyHex, err := pubKeyToHexString(pubKey[:])
		if err != nil {
			return nil, err
		}
		signedBlocks, err := getSignedBlocksByPubKey(ctx, validatorDB, pubKey)
		if err != nil {
			return nil, err
		}
		dataByPubKey[pubKey] = &ProtectionData{
			Pubkey:             pubKeyHex,
			SignedBlocks:       signedBlocks,
			SignedAttestations: nil,
		}
	}

	// Extract the signed attestations by public key.
	for _, pubKey := range attestedPublicKeys {
		pubKeyHex, err := pubKeyToHexString(pubKey[:])
		if err != nil {
			return nil, err
		}
		signedAttestations, err := getSignedAttestationsByPubKey(ctx, validatorDB, pubKey)
		if err != nil {
			return nil, err
		}
		if _, ok := dataByPubKey[pubKey]; ok {
			dataByPubKey[pubKey].SignedAttestations = signedAttestations
		} else {
			dataByPubKey[pubKey] = &ProtectionData{
				Pubkey:             pubKeyHex,
				SignedBlocks:       nil,
				SignedAttestations: signedAttestations,
			}
		}
	}

	// Next we turn our map into a slice as expected by the EIP-3076 JSON standard.
	dataList := make([]*ProtectionData, 0)
	for _, item := range dataByPubKey {
		if item.SignedAttestations == nil {
			item.SignedAttestations = make([]*SignedAttestation, 0)
		}
		if item.SignedBlocks == nil {
			item.SignedBlocks = make([]*SignedBlock, 0)
		}
		dataList = append(dataList, item)
	}
	sort.Slice(dataList, func(i, j int) bool {
		return strings.Compare(dataList[i].Pubkey, dataList[j].Pubkey) < 0
	})
	interchangeJSON.Data = dataList
	return interchangeJSON, nil
}

func getSignedAttestationsByPubKey(ctx context.Context, validatorDB db.Database, pubKey [48]byte) ([]*SignedAttestation, error) {
	// If a key does not have an attestation history in our database, we return nil.
	// This way, a user will be able to export their slashing protection history
	// even if one of their keys does not have a history of signed attestations.
	history, err := validatorDB.AttestationHistoryForPubKeyV2(ctx, pubKey)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get attestation history for public key")
	}
	if history == nil {
		return nil, nil
	}
	signedAttestations := make([]*SignedAttestation, 0)
	lowestEpoch, exists, err := validatorDB.LowestSignedTargetEpoch(ctx, pubKey)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get lowest signed target epoch for public key")
	}
	if !exists {
		return nil, nil
	}
	highestEpoch, err := history.GetLatestEpochWritten(ctx)
	if err != nil {
		return nil, err
	}
	for i := lowestEpoch; i <= highestEpoch; i++ {
		historyAtTarget, err := history.GetTargetData(ctx, i)
		if err != nil {
			return nil, err
		}
		if !historyAtTarget.IsEmpty() {
			var root string
			if historyAtTarget.SigningRoot != nil &&
				!bytes.Equal(historyAtTarget.SigningRoot, params.BeaconConfig().ZeroHash[:]) {
				root, err = rootToHexString(historyAtTarget.SigningRoot)
				if err != nil {
					return nil, err
				}
			}
			signedAttestations = append(signedAttestations, &SignedAttestation{
				TargetEpoch: fmt.Sprintf("%d", i),
				SourceEpoch: fmt.Sprintf("%d", historyAtTarget.Source),
				SigningRoot: root,
			})
		}
	}
	return signedAttestations, nil
}

func getSignedBlocksByPubKey(ctx context.Context, validatorDB db.Database, pubKey [48]byte) ([]*SignedBlock, error) {
	// If a key does not have a lowest or highest signed proposal history
	// in our database, we return nil. This way, a user will be able to export their
	// slashing protection history even if one of their keys does not have a history
	// of signed blocks.
	lowestSignedSlot, exists, err := validatorDB.LowestSignedProposal(ctx, pubKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	highestSignedSlot, exists, err := validatorDB.HighestSignedProposal(ctx, pubKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	signedBlocks := make([]*SignedBlock, 0)
	for i := lowestSignedSlot; i <= highestSignedSlot; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		signingRoot, exists, err := validatorDB.ProposalHistoryForSlot(ctx, pubKey, i)
		if err != nil {
			return nil, err
		}
		if exists {
			signingRootHex, err := rootToHexString(signingRoot[:])
			if err != nil {
				return nil, err
			}
			signedBlocks = append(signedBlocks, &SignedBlock{
				Slot:        fmt.Sprintf("%d", i),
				SigningRoot: signingRootHex,
			})
		}
	}
	return signedBlocks, nil
}

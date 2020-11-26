package interchangeformat

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/validator/db"
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
	dataByPubKey := make(map[[48]byte]*ProtectionData)

	// Extract the signed proposals by public keys.
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

	// Next we turn our map into a slice as expected by the EIP-3076 JSON standard.
	dataList := make([]*ProtectionData, 0)
	for _, item := range dataByPubKey {
		dataList = append(dataList, item)
	}
	interchangeJSON.Data = dataList
	return interchangeJSON, nil
}

func getSignedBlocksByPubKey(ctx context.Context, validatorDB db.Database, pubKey [48]byte) ([]*SignedBlock, error) {
	lowestSignedSlot, err := validatorDB.LowestSignedProposal(ctx, pubKey)
	if err != nil {
		return nil, err
	}
	highestSignedSlot, err := validatorDB.HighestSignedProposal(ctx, pubKey)
	if err != nil {
		return nil, err
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

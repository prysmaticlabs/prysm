package interchangeformat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
)

// ImportStandardProtectionJSON takes in EIP-3076 compliant JSON file used for slashing protection
// by eth2 validators and imports its data into Prysm's internal representation of slashing
// protection in the validator client's database. For more information, see the EIP document here:
// https://eips.ethereum.org/EIPS/eip-3076.
func ImportStandardProtectionJSON(ctx context.Context, validatorDB db.Database, r io.Reader) error {
	encodedJSON, err := ioutil.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "could not read slashing protection JSON file")
	}
	interchangeJSON := &EIPSlashingProtectionFormat{}
	if err := json.Unmarshal(encodedJSON, interchangeJSON); err != nil {
		return errors.Wrap(err, "could not unmarshal slashing protection JSON file")
	}

	// We validate the `Metadata` field of the slashing protection JSON file.
	if err := validateMetadata(ctx, validatorDB, interchangeJSON); err != nil {
		return errors.Wrap(err, "slashing protection JSON metadata was incorrect")
	}

	// We need to handle duplicate public keys in the JSON file, with potentially
	// different signing histories for both attestations and blocks. We create a map
	// of pubKey -> []*SignedAttestation and pubKey -> []*SignedBlock. In a later loop, we will
	// deduplicate and transform them into our internal format.
	signedBlocksByPubKey := make(map[[48]byte][]*SignedBlock)
	signedAttestationsByPubKey := make(map[[48]byte][]*SignedAttestation)
	for _, validatorData := range interchangeJSON.Data {
		pubKey, err := pubKeyFromHex(validatorData.Pubkey)
		if err != nil {
			return fmt.Errorf("%s is not a valid public key: %v", validatorData.Pubkey, err)
		}
		signedBlocksByPubKey[pubKey] = append(
			signedBlocksByPubKey[pubKey], validatorData.SignedBlocks...,
		)
		signedAttestationsByPubKey[pubKey] = append(
			signedAttestationsByPubKey[pubKey], validatorData.SignedAttestations...,
		)
	}

	// We now deduplicate the blocks from the prior loop by hashing each item and inserting it
	// into a map of pubKey -> []*SignedBlock and pubKey -> []*SignedAttestation.
	uniqueHashes := make(map[[32]byte]bool)
	uniqueSignedBlocksByPubKey := make(map[[48]byte][]*SignedBlock)
	uniqueSignedAttestationsByPubKey := make(map[[48]byte][]*SignedAttestation)
	for pubKey, signedBlocks := range signedBlocksByPubKey {
		for _, sBlock := range signedBlocks {
			encoded, err := json.Marshal(sBlock)
			if err != nil {
				return err
			}
			h := hashutil.Hash(encoded)
			if _, ok := uniqueHashes[h]; !ok {
				uniqueHashes[h] = true
				uniqueSignedBlocksByPubKey[pubKey] = append(uniqueSignedBlocksByPubKey[pubKey], sBlock)
				continue
			}
		}
	}
	for pubKey, signedAtts := range signedAttestationsByPubKey {
		for _, sAtt := range signedAtts {
			encoded, err := json.Marshal(sAtt)
			if err != nil {
				return err
			}
			h := hashutil.Hash(encoded)
			if _, ok := uniqueHashes[h]; !ok {
				uniqueHashes[h] = true
				uniqueSignedAttestationsByPubKey[pubKey] = append(uniqueSignedAttestationsByPubKey[pubKey], sAtt)
				continue
			}
		}
	}

	attestingHistoryByPubKey := make(map[[48]byte]kv.EncHistoryData)
	proposalHistoryByPubKey := make(map[[48]byte]kv.ProposalHistoryForPubkey)
	for pubKey, signedBlocks := range uniqueSignedBlocksByPubKey {
		// Parse and transform the signed blocks data from the JSON
		// file into the internal Prysm representation of proposal history.
		proposalHistory, err := parseSignedBlocks(ctx, signedBlocks)
		if err != nil {
			return errors.Wrapf(err, "could not parse signed blocks in JSON file for key %#x", pubKey)
		}
		proposalHistoryByPubKey[pubKey] = *proposalHistory
	}

	for pubKey, signedAtts := range uniqueSignedAttestationsByPubKey {
		// Parse and transform the signed attestation data from the JSON
		// file into the internal Prysm representation of attesting history.
		attestingHistory, err := parseSignedAttestations(ctx, signedAtts)
		if err != nil {
			return errors.Wrapf(err, "could not parse signed attestations in JSON file for key %#x", pubKey)
		}
		attestingHistoryByPubKey[pubKey] = *attestingHistory
	}

	// We save the histories to disk as atomic operations, ensuring that this only occurs
	// until after we successfully parse all data from the JSON file. If there is any error
	// in parsing the JSON proposal and attesting histories, we will not reach this point.
	if err = validatorDB.SaveProposalHistoryForPubKeysV2(ctx, proposalHistoryByPubKey); err != nil {
		return errors.Wrap(err, "could not save proposal history from imported JSON to database")
	}
	if err := validatorDB.SaveAttestationHistoryForPubKeysV2(ctx, attestingHistoryByPubKey); err != nil {
		return errors.Wrap(err, "could not save attesting history from imported JSON to database")
	}
	return nil
}

func validateMetadata(ctx context.Context, validatorDB db.Database, interchangeJSON *EIPSlashingProtectionFormat) error {
	// We need to ensure the version in the metadata field matches the one we support.
	version := interchangeJSON.Metadata.InterchangeFormatVersion
	if version != INTERCHANGE_FORMAT_VERSION {
		return fmt.Errorf(
			"slashing protection JSON version '%s' is not supported, wanted '%s'",
			version,
			INTERCHANGE_FORMAT_VERSION,
		)
	}

	// We need to verify the genesis validators root matches that of our chain data, otherwise
	// the imported slashing protection JSON was created on a different chain.
	// TODO(#7813): Add this check, very important!
	return nil
}

func parseSignedBlocks(ctx context.Context, signedBlocks []*SignedBlock) (*kv.ProposalHistoryForPubkey, error) {
	proposals := make([]kv.Proposal, len(signedBlocks))
	for i, proposal := range signedBlocks {
		slot, err := uint64FromString(proposal.Slot)
		if err != nil {
			return nil, fmt.Errorf("%d is not a valid slot: %v", slot, err)
		}
		var signingRoot [32]byte
		// Signing roots are optional in the standard JSON file.
		if proposal.SigningRoot != "" {
			signingRoot, err = rootFromHex(proposal.SigningRoot)
			if err != nil {
				return nil, fmt.Errorf("%#x is not a valid root: %v", signingRoot, err)
			}
		}
		proposals[i] = kv.Proposal{
			Slot:        slot,
			SigningRoot: signingRoot[:],
		}
	}
	return &kv.ProposalHistoryForPubkey{
		Proposals: proposals,
	}, nil
}

func parseSignedAttestations(ctx context.Context, atts []*SignedAttestation) (*kv.EncHistoryData, error) {
	attestingHistory := kv.NewAttestationHistoryArray(0)
	highestEpochWritten := uint64(0)
	var err error
	for _, attestation := range atts {
		target, err := uint64FromString(attestation.TargetEpoch)
		if err != nil {
			return nil, fmt.Errorf("%d is not a valid epoch: %v", target, err)
		}
		// Keep track of the highest epoch written from the imported JSON.
		if target > highestEpochWritten {
			highestEpochWritten = target
		}
		source, err := uint64FromString(attestation.SourceEpoch)
		if err != nil {
			return nil, fmt.Errorf("%d is not a valid epoch: %v", source, err)
		}
		var signingRoot [32]byte
		// Signing roots are optional in the standard JSON file.
		if attestation.SigningRoot != "" {
			signingRoot, err = rootFromHex(attestation.SigningRoot)
			if err != nil {
				return nil, fmt.Errorf("%#x is not a valid root: %v", signingRoot, err)
			}
		}
		attestingHistory, err = attestingHistory.SetTargetData(
			ctx, target, &kv.HistoryData{Source: source, SigningRoot: signingRoot[:]},
		)
		if err != nil {
			return nil, errors.Wrap(err, "could not set target data for attesting history")
		}
	}
	attestingHistory, err = attestingHistory.SetLatestEpochWritten(ctx, highestEpochWritten)
	if err != nil {
		return nil, errors.Wrap(err, "could not set latest epoch written")
	}
	return &attestingHistory, nil
}

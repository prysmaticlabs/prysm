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
	if interchangeJSON.Data == nil {
		log.Warn("No slashing protection data to import")
		return nil
	}

	// We validate the `Metadata` field of the slashing protection JSON file.
	if err := ValidateMetadata(ctx, validatorDB, interchangeJSON); err != nil {
		return errors.Wrap(err, "slashing protection JSON metadata was incorrect")
	}

	// We need to handle duplicate public keys in the JSON file, with potentially
	// different signing histories for both attestations and blocks.
	signedBlocksByPubKey, err := ParseUniqueSignedBlocksByPubKey(interchangeJSON.Data)
	if err != nil {
		return errors.Wrap(err, "could not parse unique entries for blocks by public key")
	}
	signedAttsByPubKey, err := ParseUniqueSignedAttestationsByPubKey(interchangeJSON.Data)
	if err != nil {
		return errors.Wrap(err, "could not parse unique entries for attestations by public key")
	}

	attestingHistoryByPubKey := make(map[[48]byte]kv.EncHistoryData)
	proposalHistoryByPubKey := make(map[[48]byte]kv.ProposalHistoryForPubkey)
	for pubKey, signedBlocks := range signedBlocksByPubKey {
		// Transform the processed signed blocks data from the JSON
		// file into the internal Prysm representation of proposal history.
		proposalHistory, err := transformSignedBlocks(ctx, signedBlocks)
		if err != nil {
			return errors.Wrapf(err, "could not parse signed blocks in JSON file for key %#x", pubKey)
		}
		proposalHistoryByPubKey[pubKey] = *proposalHistory
	}

	for pubKey, signedAtts := range signedAttsByPubKey {
		// Transform the processed signed attestation data from the JSON
		// file into the internal Prysm representation of attesting history.
		attestingHistory, err := transformSignedAttestations(ctx, signedAtts)
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

func ValidateMetadata(ctx context.Context, validatorDB db.Database, interchangeJSON *EIPSlashingProtectionFormat) error {
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

// We create a map of pubKey -> []*SignedBlock. Then, we keep a map of observed hashes of
// signed blocks. If we observe a new hash, we insert those signed blocks for processing.
func ParseUniqueSignedBlocksByPubKey(data []*ProtectionData) (map[[48]byte][]*SignedBlock, error) {
	seenHashes := make(map[[32]byte]bool)
	signedBlocksByPubKey := make(map[[48]byte][]*SignedBlock)
	for _, validatorData := range data {
		pubKey, err := pubKeyFromHex(validatorData.Pubkey)
		if err != nil {
			return nil, fmt.Errorf("%s is not a valid public key: %v", validatorData.Pubkey, err)
		}
		for _, sBlock := range validatorData.SignedBlocks {
			if sBlock == nil {
				continue
			}
			encoded, err := json.Marshal(sBlock)
			if err != nil {
				return nil, err
			}
			// Namespace the hash by the public key and the encoded block.
			h := hashutil.Hash(append(pubKey[:], encoded...))
			if _, ok := seenHashes[h]; ok {
				continue
			}
			seenHashes[h] = true
			signedBlocksByPubKey[pubKey] = append(signedBlocksByPubKey[pubKey], sBlock)
		}
	}
	return signedBlocksByPubKey, nil
}

// We create a map of pubKey -> []*SignedAttestation. Then, we keep a map of observed hashes of
// signed attestations. If we observe a new hash, we insert those signed attestations for processing.
func ParseUniqueSignedAttestationsByPubKey(data []*ProtectionData) (map[[48]byte][]*SignedAttestation, error) {
	seenHashes := make(map[[32]byte]bool)
	signedAttestationsByPubKey := make(map[[48]byte][]*SignedAttestation)
	for _, validatorData := range data {
		pubKey, err := pubKeyFromHex(validatorData.Pubkey)
		if err != nil {
			return nil, fmt.Errorf("%s is not a valid public key: %v", validatorData.Pubkey, err)
		}
		for _, sAtt := range validatorData.SignedAttestations {
			if sAtt == nil {
				continue
			}
			encoded, err := json.Marshal(sAtt)
			if err != nil {
				return nil, err
			}
			// Namespace the hash by the public key and the encoded block.
			h := hashutil.Hash(append(pubKey[:], encoded...))
			if _, ok := seenHashes[h]; ok {
				continue
			}
			seenHashes[h] = true
			signedAttestationsByPubKey[pubKey] = append(signedAttestationsByPubKey[pubKey], sAtt)
		}
	}
	return signedAttestationsByPubKey, nil
}

func transformSignedBlocks(ctx context.Context, signedBlocks []*SignedBlock) (*kv.ProposalHistoryForPubkey, error) {
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

func transformSignedAttestations(ctx context.Context, atts []*SignedAttestation) (*kv.EncHistoryData, error) {
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

package interchangeformat

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/pkg/errors"
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

	attestingHistoryByPubKey := make(map[[48]byte]kv.EncHistoryData)
	proposalHistoryByPubKey := make(map[[48]byte]kv.ProposalHistoryForPubkey)
	for _, validatorData := range interchangeJSON.Data {
		pubKey, err := pubKeyFromHex(validatorData.Pubkey)
		if err != nil {
			return fmt.Errorf("%s is not a valid public key: %v", validatorData.Pubkey, err)
		}
		// Parse and transform the signed attestation data from the JSON
		// file into the internal Prysm representation of attesting history.
		attestingHistory, err := parseSignedAttestations(ctx, validatorData.SignedAttestations)
		if err != nil {
			return errors.Wrapf(err, "could not parse signed attestations in JSON file for key %#x", pubKey)
		}
		attestingHistoryByPubKey[pubKey] = *attestingHistory

		// Parse and transform the signed blocks data from the JSON
		// file into the internal Prysm representation of proposal history.
		proposalHistory, err := parseSignedBlocks(ctx, validatorData.SignedBlocks)
		if err != nil {
			return errors.Wrapf(err, "could not parse signed blocks in JSON file for key %#x", pubKey)
		}
		proposalHistoryByPubKey[pubKey] = *proposalHistory
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

func parseSignedBlocks(ctx context.Context, signedBlocks []*SignedBlock) (*kv.ProposalHistoryForPubkey, error) {
	proposals := make([]kv.Proposal, len(signedBlocks))
	for i, proposal := range signedBlocks {
		slot, err := uint64FromString(proposal.Slot)
		if err != nil {
			return nil, fmt.Errorf("%d is not a valid slot: %v", slot, err)
		}
		var signingRoot [32]byte
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
		if target > highestEpochWritten {
			highestEpochWritten = target
		}
		source, err := uint64FromString(attestation.SourceEpoch)
		if err != nil {
			return nil, fmt.Errorf("%d is not a valid epoch: %v", source, err)
		}
		var signingRoot [32]byte
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

func uint64FromString(str string) (uint64, error) {
	return strconv.ParseUint(str, 10, 64)
}

func pubKeyFromHex(str string) ([48]byte, error) {
	pubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(str, "0x"))
	if err != nil {
		return [48]byte{}, err
	}
	if len(pubKeyBytes) != 48 {
		return [48]byte{}, fmt.Errorf("public key does not correct, 48-byte length: %s", str)
	}
	var pk [48]byte
	copy(pk[:], pubKeyBytes[:48])
	return pk, nil
}

func rootFromHex(str string) ([32]byte, error) {
	rootHexBytes, err := hex.DecodeString(strings.TrimPrefix(str, "0x"))
	if err != nil {
		return [32]byte{}, err
	}
	if len(rootHexBytes) != 32 {
		return [32]byte{}, fmt.Errorf("public key does not correct, 32-byte length: %s", str)
	}
	var root [32]byte
	copy(root[:], rootHexBytes[:32])
	return root, nil
}

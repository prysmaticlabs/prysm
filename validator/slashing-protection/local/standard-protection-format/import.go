package interchangeformat

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
)

// ImportStandardProtectionJSON takes in EIP-3076 compliant JSON file used for slashing protection
// by eth2 validators and imports its data into Prysm's internal representation of slashing
// protection in the validator client's database. For more information, see the EIP document here:
// https://ethereum-magicians.org/t/eip-3076-validator-client-interchange-format-slashing-protection/4883.
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
			return err
		}
		// Parse and transform the signed attestation data from the JSON
		// file into the internal Prysm representation of attesting history.
		attestingHistory, err := parseSignedAttestations(ctx, validatorData.SignedAttestations)
		if err != nil {
			return err
		}
		attestingHistoryByPubKey[pubKey] = *attestingHistory

		// Parse and transform the signed blocks data from the JSON
		// file into the internal Prysm representation of proposal history.
		proposalHistory, err := parseSignedBlocks(ctx, validatorData.SignedBlocks)
		if err != nil {
			return err
		}
		proposalHistoryByPubKey[pubKey] = *proposalHistory
	}

	// We save the histories to disk as atomic operations, ensuring that this only occurs
	// until after we successfully parse all data from the JSON file. If there is any error
	// in parsing the JSON proposal and attesting histories, we will not reach this point.
	if err = validatorDB.SaveProposalHistoryForPubKeysV2(ctx, proposalHistoryByPubKey); err != nil {
		return err
	}
	if err := validatorDB.SaveAttestationHistoryForPubKeysV2(ctx, attestingHistoryByPubKey); err != nil {
		return err
	}
	return nil
}

func parseSignedBlocks(ctx context.Context, signedBlocks []*SignedBlock) (*kv.ProposalHistoryForPubkey, error) {
	proposals := make([]kv.Proposal, len(signedBlocks))
	for i, proposal := range signedBlocks {
		slot, err := uint64FromString(proposal.Slot)
		if err != nil {
			return nil, err
		}
		var signingRoot [32]byte
		if proposal.SigningRoot != "" {
			signingRoot, err = rootFromHex(proposal.SigningRoot)
			if err != nil {
				return nil, err
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
			return nil, err
		}
		if target > highestEpochWritten {
			highestEpochWritten = target
		}
		source, err := uint64FromString(attestation.SourceEpoch)
		if err != nil {
			return nil, err
		}
		var signingRoot [32]byte
		if attestation.SigningRoot != "" {
			signingRoot, err = rootFromHex(attestation.SigningRoot)
			if err != nil {
				return nil, err
			}
		}
		attestingHistory, err = attestingHistory.SetTargetData(
			ctx, target, &kv.HistoryData{Source: source, SigningRoot: signingRoot[:]},
		)
		if err != nil {
			return nil, err
		}
	}
	fmt.Println("Highest epoch written", highestEpochWritten)
	attestingHistory, err = attestingHistory.SetLatestEpochWritten(ctx, highestEpochWritten)
	if err != nil {
		return nil, err
	}
	return &attestingHistory, nil
}

func uint64FromString(str string) (uint64, error) {
	return strconv.ParseUint(str, 10, 64)
}

func pubKeyFromHex(str string) ([48]byte, error) {
	pubKeyBytes, err := hex.DecodeString(str)
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
	rootHexBytes, err := hex.DecodeString(str)
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

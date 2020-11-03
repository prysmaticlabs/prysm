package kv

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/prysmaticlabs/prysm/validator/db/iface/interchange"
)

// ImportInterchangeData unmarshals the data interchange blob into the protection db .
func (store *Store) ImportInterchangeData(ctx context.Context, j []byte) error {
	difJSON := new(interchange.PlainDataInterchangeFormat)
	err := json.Unmarshal(j, &difJSON)
	if err != nil {
		return err
	}
	attesterHistoryByPubKey := make(map[[48]byte]*EncHistoryData)
	for _, validatorData := range difJSON.Data {
		pubkey, err := hex.DecodeString(validatorData.Pubkey)
		if err != nil {
			return err
		}
		if len(pubkey) != 48 {
			return fmt.Errorf("public key is not of the right length: %s", validatorData.Pubkey)
		}
		var pk [48]byte
		copy(pk[:], pubkey[:48])
		ehd := NewAttestationHistoryArray(0)
		lew := uint64(0)
		for _, attestation := range validatorData.SignedAttestations {
			target, err := strconv.ParseUint(attestation.TargetEpoch, 10, 64)
			if err != nil {
				return err
			}
			if target > lew {
				lew = target
			}
			source, err := strconv.ParseUint(attestation.SourceEpoch, 10, 64)
			if err != nil {
				return err
			}
			var sr []byte
			if attestation.SigningRoot != "" {
				sr, err = hex.DecodeString(attestation.SigningRoot)
				if err != nil {
					return err
				}
			}
			ehd, err = ehd.SetTargetData(ctx, target, &HistoryData{Source: source, SigningRoot: sr})
			if err != nil {
				return err
			}
		}
		ehd, err = ehd.SetLatestEpochWritten(ctx, lew)
		if err != nil {
			return err
		}
		attesterHistoryByPubKey[pk] = ehd
		for _, proposals := range validatorData.SignedBlocks {
			slot, err := strconv.ParseUint(proposals.Slot, 10, 64)
			if err != nil {
				return err
			}
			var sr []byte
			if proposals.SigningRoot != "" {
				sr, err = hex.DecodeString(proposals.SigningRoot)
				if err != nil {
					return err
				}
			}
			err = store.SaveProposalHistoryForSlot(ctx, pubkey, slot, sr)
			if err != nil {
				return err
			}
		}
	}
	err = store.SaveAttestationHistoryForPubKeysV2(ctx, attesterHistoryByPubKey)
	if err != nil {
		return err
	}

	return nil
}

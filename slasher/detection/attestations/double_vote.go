package attestations

import (
	"bytes"
	"fmt"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// DoubleVotes looks up db for slashable attesting data that were preformed by the same validator.
func (d *AttDetector) DoubleVotes(
	validatorIdx uint64,
	dataRoot []byte,
	origAtt *ethpb.IndexedAttestation,
) ([]*ethpb.AttesterSlashing, error) {
	idxAtts, err := d.slashingDetector.SlasherDB.IdxAttsForTargetFromID(origAtt.Data.Target.Epoch, validatorIdx)
	if err != nil {
		return nil, err
	}
	if idxAtts == nil || len(idxAtts) == 0 {
		return nil, fmt.Errorf("can't check nil indexed attestation for double vote")
	}

	var idxAttsToSlash []*ethpb.IndexedAttestation
	for _, att := range idxAtts {
		if att.Data == nil {
			continue
		}
		root, err := hashutil.HashProto(att.Data)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(root[:], dataRoot) {
			idxAttsToSlash = append(idxAttsToSlash, att)
		}
	}

	var as []*ethpb.AttesterSlashing
	for _, idxAtt := range idxAttsToSlash {
		as = append(as, &ethpb.AttesterSlashing{
			Attestation_1: origAtt,
			Attestation_2: idxAtt,
		})
	}
	return as, nil
}

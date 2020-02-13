package attestations

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// DetectAttestationForSlashings returns an attester slashing if the attestation submitted
// is a slashable vote.
func (d *AttDetector) DetectAttestationForSlashings(ctx context.Context, req *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error) {
	//TODO(#3133): add signature validation
	if req.Data == nil {
		return nil, fmt.Errorf("cant hash nil data in indexed attestation")
	}
	indices := req.AttestingIndices
	root, err := hashutil.HashProto(req.Data)
	if err != nil {
		return nil, err
	}
	var attSlashingResp []*ethpb.AttesterSlashing
	attSlashings := make(chan []*ethpb.AttesterSlashing, len(indices))
	errorChans := make(chan error, len(indices))
	var wg sync.WaitGroup
	lastIdx := int64(-1)
	for _, idx := range indices {
		if int64(idx) <= lastIdx {
			return nil, fmt.Errorf("indexed attestation contains repeated or non sorted ids")
		}
		wg.Add(1)
		go func(idx uint64, root [32]byte, req *ethpb.IndexedAttestation) {
			atts, err := d.DoubleVotes(idx, root[:], req)
			if err != nil {
				errorChans <- err
				wg.Done()
				return
			}
			if atts != nil && len(atts) > 0 {
				attSlashings <- atts
			}
			atts, err = d.DetectSurroundVotes(ctx, idx, req)
			if err != nil {
				errorChans <- err
				wg.Done()
				return
			}
			if atts != nil && len(atts) > 0 {
				attSlashings <- atts
			}
			wg.Done()
			return
		}(idx, root, req)
	}
	wg.Wait()
	close(errorChans)
	close(attSlashings)
	for e := range errorChans {
		if err != nil {
			err = fmt.Errorf(err.Error() + " : " + e.Error())
			continue
		}
		err = e
	}
	for atts := range attSlashings {
		attSlashingResp = append(attSlashingResp, atts...)
	}
	return attSlashingResp, err
}

// UpdateSpanMaps updates and load all span maps from db.
func (d *AttDetector) UpdateSpanMaps(ctx context.Context, req *ethpb.IndexedAttestation) error {
	indices := req.AttestingIndices
	lastIdx := int64(-1)
	var wg sync.WaitGroup
	er := make(chan error, len(indices))
	for _, idx := range indices {
		if int64(idx) <= lastIdx {
			er <- fmt.Errorf("indexed attestation contains repeated or non sorted ids")
		}
		wg.Add(1)
		go func(i uint64) {
			spanMap, err := d.slashingDetector.SlasherDB.ValidatorSpansMap(i)
			if err != nil {
				er <- err
				wg.Done()
				return
			}
			if req.Data == nil {
				log.Trace("Got indexed attestation with no data")
				wg.Done()
				return
			}
			_, spanMap, err = d.DetectSurroundingAttestation(ctx, req.Data.Source.Epoch, req.Data.Target.Epoch, i, spanMap)
			if err != nil {
				er <- err
				wg.Done()
				return
			}
			_, spanMap, err = d.DetectSurroundedAttestations(ctx, req.Data.Source.Epoch, req.Data.Target.Epoch, i, spanMap)
			if err != nil {
				er <- err
				wg.Done()
				return
			}
			if err := d.slashingDetector.SlasherDB.SaveValidatorSpansMap(i, spanMap); err != nil {
				er <- err
				wg.Done()
				return
			}
		}(idx)
		wg.Wait()
	}
	close(er)
	for e := range er {
		log.Errorf("Got error while trying to update span maps: %v", e)
	}
	return nil
}

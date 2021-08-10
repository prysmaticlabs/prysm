package sync

import (
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

const signatureVerificationInterval = 50 * time.Millisecond

const verifierLimit = 50

type signatureVerifier struct {
	set     *bls.SignatureSet
	resChan chan error
}

func (s *Service) verifierRoutine() {
	verifierBatch := make([]*signatureVerifier, 0)
	ticker := time.NewTicker(signatureVerificationInterval)
	for {
		select {
		case <-s.ctx.Done():
			// Clean up currently utilised resources.
			ticker.Stop()
			for i := 0; i < len(verifierBatch); i++ {
				verifierBatch[i].resChan <- s.ctx.Err()
			}
			return
		case sig := <-s.signatureChan:
			verifierBatch = append(verifierBatch, sig)
			if len(verifierBatch) >= verifierLimit {
				verifierBatch = verifyBatch(verifierBatch)
			}
		case <-ticker.C:
			if len(verifierBatch) > 0 {
				verifierBatch = verifyBatch(verifierBatch)
			}
		}
	}
}

func verifyBatch(verifierBatch []*signatureVerifier) []*signatureVerifier {
	aggSet := verifierBatch[0].set
	verificationErr := error(nil)

	for i := 1; i < len(verifierBatch); i++ {
		aggSet = aggSet.Join(verifierBatch[i].set)
	}
	verified, err := aggSet.Verify()
	switch {
	case err != nil:
		verificationErr = err
	case !verified:
		verificationErr = errors.New("batch signature verification failed")
	}
	for i := 0; i < len(verifierBatch); i++ {
		verifierBatch[i].resChan <- verificationErr
	}
	verifierBatch = []*signatureVerifier{}
	return verifierBatch
}

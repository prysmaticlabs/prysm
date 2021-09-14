package sync

import (
	"context"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

const signatureVerificationInterval = 50 * time.Millisecond

const verifierLimit = 50

type signatureVerifier struct {
	set     *bls.SignatureSet
	resChan chan error
}

// A routine that runs in the background to perform batch
// verifications of incoming messages from gossip.
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
				verifyBatch(verifierBatch)
				verifierBatch = []*signatureVerifier{}
			}
		case <-ticker.C:
			if len(verifierBatch) > 0 {
				verifyBatch(verifierBatch)
				verifierBatch = []*signatureVerifier{}
			}
		}
	}
}

func (s *Service) validateWithBatchVerifier(ctx context.Context, message string, set *bls.SignatureSet) pubsub.ValidationResult {
	ctx, span := trace.StartSpan(ctx, "sync.validateWithBatchVerifier")
	defer span.End()

	resChan := make(chan error)
	verificationSet := &signatureVerifier{set: set.Copy(), resChan: resChan}
	s.signatureChan <- verificationSet

	resErr := <-resChan
	close(resChan)
	// If verification fails we fallback to individual verification
	// of each signature set.
	if resErr != nil {
		log.WithError(resErr).Tracef("Could not perform batch verification of %s", message)
		verified, err := set.Verify()
		if err != nil {
			log.WithError(err).Debugf("Could not verify %s", message)
			traceutil.AnnotateError(span, err)
			return pubsub.ValidationReject
		}
		if !verified {
			log.Debugf("Verification of %s failed", message)
			traceutil.AnnotateError(span, err)
			return pubsub.ValidationReject
		}
	}
	return pubsub.ValidationAccept
}

func verifyBatch(verifierBatch []*signatureVerifier) {
	if verifierBatch == nil || len(verifierBatch) == 0 {
		return
	}
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
}

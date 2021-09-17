package sync

import (
	"context"
	"testing"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestValidateWithBatchVerifier(t *testing.T) {
	_, keys, err := testutil.DeterministicDepositsAndKeys(10)
	assert.NoError(t, err)
	sig := keys[0].Sign(make([]byte, 32))
	badSig := keys[1].Sign(make([]byte, 32))
	validSet := &bls.SignatureSet{
		Messages:   [][32]byte{{}},
		PublicKeys: []bls.PublicKey{keys[0].PublicKey()},
		Signatures: [][]byte{sig.Marshal()},
	}
	invalidSet := &bls.SignatureSet{
		Messages:   [][32]byte{{}},
		PublicKeys: []bls.PublicKey{keys[0].PublicKey()},
		Signatures: [][]byte{badSig.Marshal()},
	}
	tests := []struct {
		name          string
		message       string
		set           *bls.SignatureSet
		preFilledSets []*bls.SignatureSet
		want          pubsub.ValidationResult
	}{
		{
			name:    "empty queue",
			message: "random",
			set:     validSet,
			want:    pubsub.ValidationAccept,
		},
		{
			name:    "invalid set",
			message: "random",
			set:     invalidSet,
			want:    pubsub.ValidationReject,
		},
		{
			name:          "invalid set in routine with valid set",
			message:       "random",
			set:           validSet,
			preFilledSets: []*bls.SignatureSet{invalidSet},
			want:          pubsub.ValidationAccept,
		},
		{
			name:          "valid set in routine with invalid set",
			message:       "random",
			set:           invalidSet,
			preFilledSets: []*bls.SignatureSet{validSet},
			want:          pubsub.ValidationReject,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			svc := &Service{
				ctx:           ctx,
				cancel:        cancel,
				signatureChan: make(chan *signatureVerifier, verifierLimit),
			}
			go svc.verifierRoutine()
			for _, st := range tt.preFilledSets {
				svc.signatureChan <- &signatureVerifier{set: st, resChan: make(chan error, 10)}
			}
			if got := svc.validateWithBatchVerifier(context.Background(), tt.message, tt.set); got != tt.want {
				t.Errorf("validateWithBatchVerifier() = %v, want %v", got, tt.want)
			}
			cancel()
		})
	}
}

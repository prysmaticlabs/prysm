package payloadattestation

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives" 
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

func TestVerifyCurrentSlot(t *testing.T){
	clock := &startup.Clock{}
	verifier := &PayloadVerifier{
		Resources: Resources{
			clock: clock,
		},
	}
	assert.Equal(t, verifier.VerifyCurrentSlot(), ErrMismatchCurrentSlot)
}

func TestVerifyKnownPayloadStatus(t *testing.T){
	ptcstatus := primitives.PTCStatus(4)
	verifier := &PayloadVerifier{
		payloadAtt: ReadOnlyPayloadAtt{
			message: &ethpb.PayloadAttestationMessage{
				Data: &ethpb.PayloadAttestationData{
					PayloadStatus:ptcstatus,
				},
			},
		},
	}
	assert.Equal(t, verifier.VerifyKnownPayloadStatus(), ErrUnknownPayloadStatus)
}
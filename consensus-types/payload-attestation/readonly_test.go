package payloadattestation

import (
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)


func TestValidatePayload(t *testing.T){
	tests := []struct {
		name  string
		bfunc func(t *testing.T) *ethpb.PayloadAttestationMessage
		err   error
	}{
		{
			name: "nil PayloadAttestationMessage",
			bfunc: func(t *testing.T) *ethpb.PayloadAttestationMessage {
				return nil
			},
			err:  errNilPayloadMessage,
		},
		{
			name: "nil data",
			bfunc: func(t *testing.T) *ethpb.PayloadAttestationMessage {
				return &ethpb.PayloadAttestationMessage{
					Data: nil,
				}
			},
			err:  errNilPayloadData,
		},
		{
			name: "nil signature",
			bfunc: func(t *testing.T) *ethpb.PayloadAttestationMessage {
				return &ethpb.PayloadAttestationMessage{
					Signature: nil,
				}
			},
			err:  errMissingPayloadSignature,
		},
	}
		for _, tt := range tests {
			t.Run(tt.name+" ReadOnlyPayloadAtt", func(t *testing.T) {
				m := tt.bfunc(t)
				err := validatePayload(m)
				if tt.err != nil {
					require.ErrorIs(t, err, tt.err)
				} else {
					RoMess,err:=NewReadOnly(m)
					assert.Equal(t,RoMess.message.Data,m.Data)
					assert.Equal(t,RoMess.message.Signature,m.Signature)
					require.NoError(t, err)
				}
			})
		}
}

func TestValidatorIndex(t *testing.T){
	ValIdx:=primitives.ValidatorIndex(1)
	m:=&ReadOnlyPayloadAtt{
		message: &ethpb.PayloadAttestationMessage{
		ValidatorIndex: ValIdx,
		},
	}
	assert.Equal(t,ValIdx,m.ValidatorIndex())
}
func TestSignature(t *testing.T){
	Sig:=[]byte{1}
	m:=&ReadOnlyPayloadAtt{
		message: &ethpb.PayloadAttestationMessage{
		Signature: Sig,
		},
	}
	assert.Equal(t,Sig,m.Signature())
}

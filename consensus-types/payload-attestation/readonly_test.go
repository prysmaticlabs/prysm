package payloadattestation

import (
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)


func TestValidatePayload(t *testing.T){
	tests := []struct {
		name  string
		bfunc func(t *testing.T) *ethpb.PayloadAttestationMessage
		wanterr   error
	}{
		{
			name: "nil PayloadAttestationMessage",
			bfunc: func(t *testing.T) *ethpb.PayloadAttestationMessage {
				return nil
			},
			wanterr:  errNilPayloadMessage,
		},
		{
			name: "nil data",
			bfunc: func(t *testing.T) *ethpb.PayloadAttestationMessage {
				return &ethpb.PayloadAttestationMessage{
					Data: nil,
				}
			},
			wanterr:  errNilPayloadData,
		},
		{
			name: "nil signature",
			bfunc: func(t *testing.T) *ethpb.PayloadAttestationMessage {
				return &ethpb.PayloadAttestationMessage{
					Signature: nil,
				}
			},
			wanterr:  errMissingPayloadSignature,
		},
		{
			name: "Correct PayloadAttestationMessage",
			bfunc: func(t *testing.T) *ethpb.PayloadAttestationMessage {
				return &ethpb.PayloadAttestationMessage{
					Signature: make([]byte, fieldparams.BLSSignatureLength),
					Data: &ethpb.PayloadAttestationData{
					BeaconBlockRoot: make([]byte, fieldparams.RootLength),
					},
				}
			},
			wanterr:  nil,
		},
	}
		for _, tt := range tests {
			t.Run(tt.name+" ReadOnlyPayloadAtt", func(t *testing.T) {
				m := tt.bfunc(t)
				err := validatePayload(m)
				if tt.wanterr != nil {
					require.ErrorIs(t, err, tt.wanterr)
				}else {
					roMess,err:=NewReadOnly(m)
					require.NoError(t, err)
					assert.Equal(t,roMess.message.Data,m.Data)
					assert.Equal(t,roMess.message.Signature,m.Signature)
				}
			})
		}
}

func TestValidatorIndex(t *testing.T){
	valIdx:=primitives.ValidatorIndex(1)
	m:=&ReadOnlyPayloadAtt{
		message: &ethpb.PayloadAttestationMessage{
		ValidatorIndex: valIdx,
		},
	}
	assert.Equal(t,valIdx,m.ValidatorIndex())
}

func TestSignature(t *testing.T){
	sig:=make([]byte,fieldparams.BLSSignatureLength)
	m:=&ReadOnlyPayloadAtt{
		message: &ethpb.PayloadAttestationMessage{
		Signature: sig[:],
		},
	}
	assert.Equal(t,sig,m.Signature())
}
func TestBeaconBlockRoot(t *testing.T){
	root := make([]byte,fieldparams.RootLength)
	m:=&ReadOnlyPayloadAtt{
		message: &ethpb.PayloadAttestationMessage{
			Data: &ethpb.PayloadAttestationData{
				BeaconBlockRoot: root[:],
			},
		},
	}
	assert.Equal(t,root,m.BeaconBlockRoot())
}

func TestSlot(t *testing.T){
	slot:=primitives.Slot(1)
	m:=&ReadOnlyPayloadAtt{
		message: &ethpb.PayloadAttestationMessage{
			Data: &ethpb.PayloadAttestationData{
				Slot: slot,
			},
		},
	}
	assert.Equal(t,slot,m.Slot())
}

func TestPayloadStatus(t *testing.T){
	for i:=0;i<4;i++ {
	status:=primitives.PTCStatus(i)
	m:=&ReadOnlyPayloadAtt{
		message: &ethpb.PayloadAttestationMessage{
			Data: &ethpb.PayloadAttestationData{
				PayloadStatus:status,
			},
		},
	}
		require.NoError(t,validatePayload(m.message))
		assert.Equal(t,status,m.PayloadStatus())
	}
}

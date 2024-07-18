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
		{
			name: "Correct PayloadAttestationMessage",
			bfunc: func(t *testing.T) *ethpb.PayloadAttestationMessage {
				return &ethpb.PayloadAttestationMessage{
					Signature: make([]byte, 96),
					Data: &ethpb.PayloadAttestationData{
					BeaconBlockRoot: make([]byte, fieldparams.RootLength),
					},
				}
			},
			err:  nil,
		},
	}
		for _, tt := range tests {
			t.Run(tt.name+" ReadOnlyPayloadAtt", func(t *testing.T) {
				m := tt.bfunc(t)
				err := validatePayload(m)
				if tt.err != nil {
					require.ErrorIs(t, err, tt.err)
				}else {
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
	Sig:=[96]byte{1}
	m:=&ReadOnlyPayloadAtt{
		message: &ethpb.PayloadAttestationMessage{
		Signature: Sig[:],
		},
	}
	assert.Equal(t,Sig,m.Signature())
}
func TestBeaconBlockRoot(t *testing.T){
	root := [32]byte{1}
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

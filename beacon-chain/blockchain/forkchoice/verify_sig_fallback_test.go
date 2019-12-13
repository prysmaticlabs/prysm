package forkchoice

import (
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestConvertToIndexed_OK(t *testing.T) {
	validators := make([]*ethpb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		Slot:        5,
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	tests := []struct {
		aggregationBitfield      bitfield.Bitlist
		custodyBitfield          bitfield.Bitlist
		wantedCustodyBit0Indices []uint64
		wantedCustodyBit1Indices []uint64
	}{
		{
			aggregationBitfield:      bitfield.Bitlist{0x07},
			custodyBitfield:          bitfield.Bitlist{0x05},
			wantedCustodyBit0Indices: []uint64{4},
			wantedCustodyBit1Indices: []uint64{30},
		},
		{
			aggregationBitfield:      bitfield.Bitlist{0x07},
			custodyBitfield:          bitfield.Bitlist{0x06},
			wantedCustodyBit0Indices: []uint64{30},
			wantedCustodyBit1Indices: []uint64{4},
		},
		{
			aggregationBitfield:      bitfield.Bitlist{0x07},
			custodyBitfield:          bitfield.Bitlist{0x07},
			wantedCustodyBit0Indices: []uint64{},
			wantedCustodyBit1Indices: []uint64{4, 30},
		},
	}

	attestation := &ethpb.Attestation{
		Signature: []byte("signed"),
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
	}
	for _, tt := range tests {
		attestation.AggregationBits = tt.aggregationBitfield
		attestation.CustodyBits = tt.custodyBitfield
		wanted := &ethpb.IndexedAttestation{
			CustodyBit_0Indices: tt.wantedCustodyBit0Indices,
			CustodyBit_1Indices: tt.wantedCustodyBit1Indices,
			Data:                attestation.Data,
			Signature:           attestation.Signature,
		}
		ia, err := convertToIndexed(context.Background(), state, attestation)
		if err != nil {
			t.Errorf("failed to convert attestation to indexed attestation: %v", err)
		}
		if !reflect.DeepEqual(wanted, ia) {
			t.Error("convert attestation to indexed attestation didn't result as wanted")
		}
	}
}


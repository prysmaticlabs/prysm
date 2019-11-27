package db_test

import (
	"fmt"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

func TestNewContainerFromAttestations(t *testing.T) {
	atts := []*ethpb.Attestation{
		{
			AggregationBits: bitfield.Bitlist{0b00000001, 0b1},
			Signature:       bls.NewAggregateSignature().Marshal(),
			Data:            &ethpb.AttestationData{},
		},
	}

	ac := dbpb.NewContainerFromAttestations(atts)

	if len(ac.SignaturePairs) != 1 {
		t.Errorf("wrong length of pairs. wanted 1 got %d", len(ac.SignaturePairs))
	}
}

func TestAttestationContainer_Contains(t *testing.T) {
	tests := []struct {
		input    []bitfield.Bitlist
		contains bitfield.Bitlist
		want     bool
	}{
		{
			input: []bitfield.Bitlist{
				{0b10000001, 0b1},
				{0b10000010, 0b1},
				{0b10000100, 0b1},
				{0b10010001, 0b1},
			},
			contains: bitfield.Bitlist{0b00000001, 0b1},
			want:     true,
		},
		{
			input: []bitfield.Bitlist{
				{0b10000001, 0b1},
				{0b10000010, 0b1},
				{0b10000100, 0b1},
				{0b10010001, 0b1},
			},
			contains: bitfield.Bitlist{0b00001000, 0b1},
			want:     false,
		},
		{
			input: []bitfield.Bitlist{
				{0b10000001, 0b1},
				{0b10000010, 0b1},
				{0b10000100, 0b1},
				{0b10010001, 0b1},
			},
			contains: bitfield.Bitlist{0b01000000, 0b1},
			want:     false,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			var sp []*dbpb.AttestationContainer_SignaturePair
			for _, input := range tt.input {
				sp = append(sp, &dbpb.AttestationContainer_SignaturePair{
					AggregationBits: input,
					Signature:       bls.NewAggregateSignature().Marshal(),
				})
			}

			ac := &dbpb.AttestationContainer{
				SignaturePairs: sp,
			}

			if ac.Contains(&ethpb.Attestation{AggregationBits: tt.contains}) != tt.want {
				t.Errorf("AttestationContainer.Contains returned wrong value, got %v wanted %v", !tt.want, tt.want)
			}
		})
	}
}

func TestAttestationContainer_ToAttestations(t *testing.T) {
	data := &ethpb.AttestationData{BeaconBlockRoot: []byte("foo")}
	sig := bls.NewAggregateSignature().Marshal()
	ac := &dbpb.AttestationContainer{
		Data: data,
		SignaturePairs: []*dbpb.AttestationContainer_SignaturePair{
			{AggregationBits: bitfield.Bitlist{0b00000001, 0b1}, Signature: sig},
			{AggregationBits: bitfield.Bitlist{0b00000010, 0b1}, Signature: sig},
		},
	}

	want := []*ethpb.Attestation{
		{
			Data:            data,
			AggregationBits: bitfield.Bitlist{0b00000001, 0b1},
			Signature:       sig,
			CustodyBits:     bitfield.NewBitlist(8),
		},
		{
			Data:            data,
			AggregationBits: bitfield.Bitlist{0b00000010, 0b1},
			Signature:       sig,
			CustodyBits:     bitfield.NewBitlist(8),
		},
	}

	if !reflect.DeepEqual(ac.ToAttestations(), want) {
		t.Error("ToAttestations returned the wrong value")
	}
}

func TestAttestationContainer_InsertAttestation(t *testing.T) {
	tests := []struct {
		input  []*ethpb.Attestation
		output []*dbpb.AttestationContainer_SignaturePair
	}{
		{
			input: []*ethpb.Attestation{
				{
					AggregationBits: bitfield.Bitlist{0b00000011, 0b1},
				},
				{
					AggregationBits: bitfield.Bitlist{0b00000101, 0b1},
				},
			},
			output: []*dbpb.AttestationContainer_SignaturePair{
				{
					AggregationBits: bitfield.Bitlist{0b00000011, 0b1},
				},
				{
					AggregationBits: bitfield.Bitlist{0b00000101, 0b1},
				},
			},
		},
		{
			input: []*ethpb.Attestation{
				{
					AggregationBits: bitfield.Bitlist{0b00000011, 0b1},
				},
				{
					AggregationBits: bitfield.Bitlist{0b00000111, 0b1},
				},
			},
			output: []*dbpb.AttestationContainer_SignaturePair{
				{
					AggregationBits: bitfield.Bitlist{0b00000111, 0b1},
				},
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			ac := &dbpb.AttestationContainer{}
			for _, att := range tt.input {
				ac.InsertAttestation(att)
			}
			if !reflect.DeepEqual(ac.SignaturePairs, tt.output) {
				t.Error("Signature pairs do not match expected output")
			}
		})
	}
}

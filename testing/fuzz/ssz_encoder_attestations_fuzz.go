package fuzz

import (
	"bytes"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

var buf = new(bytes.Buffer)

// SszEncoderAttestationFuzz runs network encode/decode for attestations.
func SszEncoderAttestationFuzz(b []byte) {
	params.UseMainnetConfig()
	buf.Reset()
	input := &ethpb.Attestation{}
	e := encoder.SszNetworkEncoder{}
	if err := e.DecodeGossip(b, input); err != nil {
		_ = err
		return
	}
	if _, err := e.EncodeGossip(buf, input); err != nil {
		_ = err
		return
	}
}

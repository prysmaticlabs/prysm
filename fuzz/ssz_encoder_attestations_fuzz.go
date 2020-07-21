package fuzz

import (
	"bytes"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var buf = new(bytes.Buffer)

// SszEncoderAttestationFuzz -- TODO.
func SszEncoderAttestationFuzz(b []byte)  {
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

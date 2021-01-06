package slashutil

import ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

// IsSurrounding checks if an incoming attestation, b, is surrounding
// a previous one, a, based on the eth2 slashing conditions specified
// by @protolambda https://github.com/protolambda/eth2-surround#definition.
//
//  s: source
//  t: target
//
//  a surrounds b if: s_a < s_b and t_b < t_a
//
func IsSurrounding(a, b *ethpb.IndexedAttestation) bool {
	return a.Data.Source.Epoch < b.Data.Source.Epoch && b.Data.Target.Epoch < a.Data.Target.Epoch
}

// IsSurrounded checks if an incoming attestation, b, is surrounded by
// a previous one, a, based on the eth2 slashing conditions specified
// by @protolambda https://github.com/protolambda/eth2-surround#definition.
//
//  s: source
//  t: target
//
//  b is surrounded by a if: s_a > s_b and t_b > t_a
//
func IsSurrounded(a, b *ethpb.IndexedAttestation) bool {
	return a.Data.Source.Epoch > b.Data.Source.Epoch && b.Data.Target.Epoch > a.Data.Target.Epoch
}

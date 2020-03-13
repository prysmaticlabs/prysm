package kv

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// AttCaches defines the caches used to satisfy attestation pool interface.
// These caches are KV store for various attestations
// such are unaggregated, aggregated or attestations within a block.
type AttCaches struct {
	aggregatedAtt   map[[32]byte][]*ethpb.Attestation
	unAggregatedAtt map[[32]byte]*ethpb.Attestation
	forkchoiceAtt   map[[32]byte]*ethpb.Attestation
	blockAtt        map[[32]byte][]*ethpb.Attestation
}

// NewAttCaches initializes a new attestation pool consists of multiple KV store in cache for
// various kind of attestations.
func NewAttCaches() *AttCaches {
	pool := &AttCaches{
		unAggregatedAtt: make(map[[32]byte]*ethpb.Attestation),
		aggregatedAtt:   make(map[[32]byte][]*ethpb.Attestation),
		forkchoiceAtt:   make(map[[32]byte]*ethpb.Attestation),
		blockAtt:        make(map[[32]byte][]*ethpb.Attestation),
	}

	return pool
}

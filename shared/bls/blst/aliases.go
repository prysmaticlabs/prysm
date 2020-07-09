package blst

import blst "github.com/supranational/blst/bindings/go"

// Internal types for blst.
type blstPublicKey = blst.P1Affine
type blstSignature = blst.P2Affine
type blstAggregateSignature = blst.P2Aggregate
type blstAggregatePublicKey = blst.P1Aggregate

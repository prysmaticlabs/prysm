package bls

// DomainByteLength length of domain byte array.
const DomainByteLength = 4

// CurveOrder for the BLS12-381 curve.
const CurveOrder = "52435875175126190479447740508185965837690552500527637822603658699938581184513"

// List of descriptions for different kinds of signatures
const (
	// UnknownSignature represents all signatures other than below types
	UnknownSignature string = "unknown signature"
	// AggregatedSignature represents aggregated signature produced by AggregateBatch()
	AggregatedSignature = "bls aggregated signature"
	// BlockSignature represents the block signature from block proposer
	BlockSignature = "block signature"
	// RandaoSignature represents randao specific signature
	RandaoSignature = "randao signature"
	// SelectionProof represents selection proof
	SelectionProof = "selection proof"
	// AggregatorSignature represents aggregator's signature
	AggregatorSignature = "aggregator signature"
	// AttestationSignature represents aggregated attestation signature
	AttestationSignature = "attestation signature"
	// BlsChangeSignature represents signature to BLSToExecutionChange
	BlsChangeSignature = "blschange signature"
	// SyncCommitteeSignature represents sync committee signature
	SyncCommitteeSignature = "sync committee signature"
	// SyncSelectionProof represents sync committee selection proof
	SyncSelectionProof = "sync selection proof"
	// ContributionSignature represents sync committee contributor's signature
	ContributionSignature = "sync committee contribution signature"
	// SyncAggregateSignature represents sync committee aggregator's signature
	SyncAggregateSignature = "sync committee aggregator signature"
)

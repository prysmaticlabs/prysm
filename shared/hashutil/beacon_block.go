package hashutil

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// HashBeaconBlock hashes the full block without the block signature.
// The block signature is ignored in order obtain the same block hash used
// as the "block_root" property in the block signature data.
func HashBeaconBlock(bb *pb.BeaconBlock) ([32]byte, error) {
	// Ignore the block signature by temporarily deleting it.
	sig := bb.Signature
	bb.Signature = nil
	defer func() { bb.Signature = sig }()

	return HashProto(bb)
}

// HashProposal hashes the proposal without the proposal signature.
// The proposal signature is ignored in order obtain the same proposal hash used
// as the "proposal_signed_data" property in the proposal signature data.
func HashProposal(p *pb.Proposal) ([32]byte, error) {
	// Ignore the proposal signature by temporarily deleting it.
	sig := p.Signature
	p.Signature = nil
	defer func() { p.Signature = sig }()

	return HashProto(p)
}

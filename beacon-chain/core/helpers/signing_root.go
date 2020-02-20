package helpers

import (
	"github.com/prysmaticlabs/go-ssz"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// ComputeSigningRoot computes the root of the object by calculating the root of the object domain tree.
//
// Spec pseudocode definition:
//	def compute_signing_root(ssz_object: SSZObject, domain: Domain) -> Root:
//    """
//    Return the signing root of an object by calculating the root of the object-domain tree.
//    """
//    domain_wrapped_object = SigningRoot(
//        object_root=hash_tree_root(ssz_object),
//        domain=domain,
//    )
//    return hash_tree_root(domain_wrapped_object)
func ComputeSigningRoot(object interface{}, domain []byte) ([32]byte, error) {
	objRoot, err := ssz.HashTreeRoot(object)
	if err != nil {
		return [32]byte{}, err
	}
	container := &p2ppb.SigningRoot{
		ObjectRoot: objRoot[:],
		Domain:     domain,
	}
	return ssz.HashTreeRoot(container)
}

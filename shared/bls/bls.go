/**
* File        : bls.go
* Description : Boneh-Lynn-Shacham signature scheme.
* Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
* Maintainer  : Enzo Haussecker <enzo@dfinity.org>
* Stability   : Stable
*
* This module implements the Boneh-Lynn-Shacham signature scheme. It includes
* support for aggregate signatures, threshold signatures, and ring signatures.
*/

package bls

import (
	"crypto/sha256"
	"errors"
	"math/big"
	"unsafe"
	"C"
)

/*
#cgo LDFLAGS: -lgmp -lpbc
#include <pbc.h>
#include <stdio.h>
#include <stdlib.h>
int callback(pbc_cm_t cm, void *data) {
pbc_param_init_d_gen(data, cm);
return 1;
}

int search(pbc_param_ptr params, unsigned int d, unsigned int bitlimit) {
int m = d % 4;
if (d == 0 || m == 1 || m == 2) {
	pbc_die("Discriminant must be 0 or 3 mod 4 and positive.");
}
return pbc_cm_search_d(callback, params, d, bitlimit);
}
*/

const sizeOfElement = C.size_t(unsafe.Sizeof(C.struct_element_s{}))
const sizeOfParams = C.size_t(unsafe.Sizeof(C.struct_pbc_param_s{}))
const sizeOfPairing = C.size_t(unsafe.Sizeof(C.struct_pairing_s{}))

// Element of a BLS system.
type Element struct {
	get *C.struct_element_s
}

// Params of a Pairing.
type Params struct {
	get *C.struct_pbc_param_s
}

// Pairing definition for BLS.
type Pairing struct {
	get *C.struct_pairing_s
}

// System encompassing requirements of BLS signature aggregation.
type System struct {
	pairing Pairing
	g       Element
}

// PublicKey of an actor.
type PublicKey struct {
	system System
	gx     Element
}

// PrivateKey of an actor.
type PrivateKey struct {
	system System
	x      Element
}

// Signature is the primitive BLS will work with.
type Signature = Element

// GenParamsTypeA pairing parameters. This function allocates C structures on
// the C heap using malloc. It is the responsibility of the caller to prevent
// memory leaks by arranging for the C structures to be freed. More information
// about type A pairing parameters can be found in the PBC library manual:
// https://crypto.stanford.edu/pbc/manual/ch08s03.html.
func GenParamsTypeA(rbits int, qbits int) Params {
	params := (*C.struct_pbc_param_s)(C.malloc(sizeOfParams))
	C.pbc_param_init_a_gen(params, C.int(rbits), C.int(qbits))
	return Params{params}
}

// GenParamsTypeD pairing parameters. This function allocates C structures on
// the C heap using malloc. It is the responsibility of the caller to prevent
// memory leaks by arranging for the C structures to be freed. More information
// about type D pairing parameters can be found in the PBC library manual:
// https://crypto.stanford.edu/pbc/manual/ch08s06.html.
func GenParamsTypeD(d uint, bitlimit uint) (Params, error) {
	params := (*C.struct_pbc_param_s)(C.malloc(sizeOfParams))
	if C.search(params, C.uint(d), C.uint(bitlimit)) == 0 {
		return Params{}, errors.New("bls.GenParamsTypeD: no suitable curves for this discriminant")
	}
	return Params{params}, nil
}

// GenParamsTypeF pairing parameters. This function allocates C structures on
// the C heap using malloc. It is the responsibility of the caller to prevent
// memory leaks by arranging for the C structures to be freed. More information
// about type F pairing parameters can be found in the PBC library manual:
// https://crypto.stanford.edu/pbc/manual/ch08s08.html.
func GenParamsTypeF(bits int) Params {
	params := (*C.struct_pbc_param_s)(C.malloc(sizeOfParams))
	C.pbc_param_init_f_gen(params, C.int(bits))
	return Params{params}
}

// ParamsFromBytes imports Params from the provided byte slice.
// It expects the data format exported by ToBytes. An example of Type A
// params of this form can be found in param/a.param
//
// This function allocates C structures on the C heap using malloc. It is
// the responsibility of the caller to prevent memory leaks by arranging
// for the C structures to be freed.
func ParamsFromBytes(bytes []byte) (Params, error) {
	s := C.CString(string(bytes))
	defer C.free(unsafe.Pointer(s))
	params := (*C.struct_pbc_param_s)(C.malloc(sizeOfParams))
	rv := C.pbc_param_init_set_str(params, s)
	if rv == 1 {
		return Params{}, errors.New("bls.FromBytes: failed to create params from bytes")
	}
	return Params{params}, nil
}

// GenPairing generates a pairing from the given parameters. This function allocates C
// structures on the C heap using malloc. It is the responsibility of the caller
// to prevent memory leaks by arranging for the C structures to be freed.
func GenPairing(params Params) Pairing {
	pairing := (*C.struct_pairing_s)(C.malloc(sizeOfPairing))
	C.pairing_init_pbc_param(pairing, params.get)
	return Pairing{pairing}
}

// GenSystem creates a cryptosystem from the given pairing. This function allocates C
// structures on the C heap using malloc. It is the responsibility of the caller
// to prevent memory leaks by arranging for the C structures to be freed.
func GenSystem(pairing Pairing) (System, error) {
	// Generate a cryptographically secure pseudorandom hash.
	hash, err := randomHash()
	if err != nil {
		return System{}, err
	}

	// Derive the system parameter from the pseudorandom hash.
	g := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_G2(g, pairing.get)
	C.element_from_hash(g, unsafe.Pointer(&hash[0]), sha256.Size)

	// Return the cryptosystem.
	return System{pairing, Element{g}}, nil
}

// SystemFromBytes imports a System from the provided byte slice. 
// This function allocates C structures on the C heap using malloc. It is
// the responsibility of the caller to prevent memory leaks by arranging
// for the C structures to be freed.
func SystemFromBytes(pairing Pairing, bytes []byte) (System, error) {
	n := int(C.pairing_length_in_bytes_compressed_G2(pairing.get))
	if n != len(bytes) {
		return System{}, errors.New("bls.FromBytes: system length mismatch")
	}
	g := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_G2(g, pairing.get)
	C.element_from_bytes_compressed(g, (*C.uchar)(unsafe.Pointer(&bytes[0])))
	return System{pairing, Element{g}}, nil
}

// GenKeys creates a keypair from the given cryptosystem. This function allocates C
// structures on the C heap using malloc. It is the responsibility of the caller
// to prevent memory leaks by arranging for the C structures to be freed.
func GenKeys(system System) (PublicKey, PrivateKey, error) {
	// Generate a cryptographically secure pseudorandom hash.
	hash, err := randomHash()
	if err != nil {
		return PublicKey{}, PrivateKey{}, err
	}

	// Derive the private key from the pseudorandom hash.
	x := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_Zr(x, system.pairing.get)
	C.element_from_hash(x, unsafe.Pointer(&hash[0]), sha256.Size)

	// Derive the public key from the private key.
	gx := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_G2(gx, system.pairing.get)
	C.element_pow_zn(gx, system.g.get, x)

	// Return the key pair.
	return PublicKey{system, Element{gx}}, PrivateKey{system, Element{x}}, nil
}

// GenKeyShares takes the keys from the given cryptosystem and divide each key into n
// shares such that t shares can combine signatures to recover a threshold
// signature. This function allocates C structures on the C heap using malloc.
// It is the responsibility of the caller to prevent memory leaks by arranging
// for the C structures to be freed.
func GenKeyShares(t int, n int, system System) (PublicKey, []PublicKey, PrivateKey, []PrivateKey, error) {
	// Check the threshold parameters.
	if t < 1 || n < t {
		return PublicKey{}, nil, PrivateKey{}, nil, errors.New("bls.GenKeyShares: bad threshold parameters")
	}

	// Generate a polynomial.
	coeff := make([]*C.struct_element_s, t)
	var hash [sha256.Size]byte
	var err error
	for j := range coeff {

		// Generate a cryptographically secure pseudorandom hash.
		hash, err = randomHash()
		if err != nil {
			return PublicKey{}, nil, PrivateKey{}, nil, err
		}

		// Derive a coefficient of the polynomial from the pseudorandom hash.
		coeff[j] = (*C.struct_element_s)(C.malloc(sizeOfElement))
		C.element_init_Zr(coeff[j], system.pairing.get)
		C.element_from_hash(coeff[j], unsafe.Pointer(&hash[0]), sha256.Size)

	}

	// Derive the key pair and the key shares from the polynomial.
	keys := make([]PublicKey, n+1)
	secrets := make([]PrivateKey, n+1)
	var bytes []byte
	var ij C.mpz_t
	C.mpz_init(&ij[0])
	term := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_Zr(term, system.pairing.get)
	for i := 0; i < n+1; i++ {

		// Calculate a share of the private key by evaluating the polynomial.
		secrets[i].system = system
		secrets[i].x.get = (*C.struct_element_s)(C.malloc(sizeOfElement))
		C.element_init_Zr(secrets[i].x.get, system.pairing.get)
		C.element_set0(secrets[i].x.get)
		for j := 0; j < t; j++ {
			bytes = big.NewInt(0).Exp(big.NewInt(int64(i)), big.NewInt(int64(j)), nil).Bytes()
			if len(bytes) == 0 {
				C.mpz_set_si(&ij[0], 0)
			} else {
				C.mpz_import(&ij[0], C.size_t(len(bytes)), 1, 1, 1, 0, unsafe.Pointer(&bytes[0]))
			}
			C.element_mul_mpz(term, coeff[j], &ij[0])
			C.element_add(secrets[i].x.get, secrets[i].x.get, term)
		}

		// Calculate a share of the public key by exponentiating the system parameter.
		keys[i].system = system
		keys[i].gx.get = (*C.struct_element_s)(C.malloc(sizeOfElement))
		C.element_init_G2(keys[i].gx.get, system.pairing.get)
		C.element_pow_zn(keys[i].gx.get, system.g.get, secrets[i].x.get)
	}

	// Clean up.
	for j := range coeff {
		C.element_clear(coeff[j])
	}
	C.mpz_clear(&ij[0])
	C.element_clear(term)

	// Return the key pair and the key shares.
	return keys[0], keys[1:], secrets[0], secrets[1:], nil
}

// Sign a message digest using a private key. This function allocates C
// structures on the C heap using malloc. It is the responsibility of the caller
// to prevent memory leaks by arranging for the C structures to be freed.
func Sign(hash [sha256.Size]byte, secret PrivateKey) Signature {

	// Calculate h.
	h := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_G1(h, secret.system.pairing.get)
	C.element_from_hash(h, unsafe.Pointer(&hash[0]), sha256.Size)

	// Calculate sigma.
	sigma := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_G1(sigma, secret.system.pairing.get)
	C.element_pow_zn(sigma, h, secret.x.get)

	// Clean up.
	C.element_clear(h)

	// Return the signature.
	return Element{sigma}
}

// Verify a signature on the message digest using the public key of the signer.
func Verify(signature Signature, hash [sha256.Size]byte, key PublicKey) bool {
	// Calculate the left-hand side.
	lhs := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_GT(lhs, key.system.pairing.get)
	C.element_pairing(lhs, signature.get, key.system.g.get)

	// Calculate h.
	h := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_G1(h, key.system.pairing.get)
	C.element_from_hash(h, unsafe.Pointer(&hash[0]), sha256.Size)

	// Calculate the right-hand side.
	rhs := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_GT(rhs, key.system.pairing.get)
	C.element_pairing(rhs, h, key.gx.get)

	// Equate the left and right-hand side.
	C.element_invert(rhs, rhs)
	C.element_mul(lhs, lhs, rhs)
	result := C.element_is1(lhs) == 1

	// Clean up.
	C.element_clear(h)
	C.element_clear(lhs)
	C.element_clear(rhs)

	// Return the result.
	return result
}

// Aggregate signatures using the cryptosystem. This function allocates C
// structures on the C heap using malloc. It is the responsibility of the caller
// to prevent memory leaks by arranging for the C structures to be freed.
func Aggregate(signatures []Signature, system System) (Signature, error) {
	// Check the list length.
	if len(signatures) == 0 {
		return Element{}, errors.New("bls.Aggregate: empty list")
	}

	// Calculate sigma.
	sigma := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_G1(sigma, system.pairing.get)
	C.element_set(sigma, signatures[0].get)
	t := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_G1(t, system.pairing.get)
	for i := 1; i < len(signatures); i++ {
		C.element_mul(sigma, sigma, signatures[i].get)
	}

	// Clean up.
	C.element_clear(t)

	// Return the aggregate signature.
	return Element{sigma}, nil
}

// AggregateVerify an aggregate signature on the message digests using the public keys of
// the signers.
func AggregateVerify(signature Signature, hashes [][sha256.Size]byte, keys []PublicKey) (bool, error) {
	// Check the list length.
	if len(hashes) == 0 {
		return false, errors.New("bls.AggregateVerify: empty list")
	}
	if len(hashes) != len(keys) {
		return false, errors.New("bls.AggregateVerify: list length mismatch")
	}

	// Check the uniqueness constraint.
	if !uniqueHashes(hashes) {
		return false, errors.New("bls.AggregateVerify: message digests must be distinct")
	}

	// Calculate the left-hand side.
	lhs := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_GT(lhs, keys[0].system.pairing.get)
	C.element_pairing(lhs, signature.get, keys[0].system.g.get)

	// Calculate the right-hand side.
	h := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_G1(h, keys[0].system.pairing.get)
	C.element_from_hash(h, unsafe.Pointer(&hashes[0][0]), sha256.Size)
	rhs := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_GT(rhs, keys[0].system.pairing.get)
	C.element_pairing(rhs, h, keys[0].gx.get)
	t := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_GT(t, keys[0].system.pairing.get)
	for i := 1; i < len(hashes); i++ {
		C.element_from_hash(h, unsafe.Pointer(&hashes[i][0]), sha256.Size)
		C.element_pairing(t, h, keys[i].gx.get)
		C.element_mul(rhs, rhs, t)
	}

	// Equate the left and right-hand side.
	C.element_invert(rhs, rhs)
	C.element_mul(lhs, lhs, rhs)
	result := C.element_is1(lhs) == 1

	// Clean up.
	C.element_clear(h)
	C.element_clear(lhs)
	C.element_clear(rhs)
	C.element_clear(t)

	// Return the result.
	return result, nil
}

// Threshold recovers a threshold signature from the signature shares provided by the group
// members using the cryptosystem. This function allocates C structures on the C
// heap using malloc. It is the responsibility of the caller to prevent memory
// leaks by arranging for the C structures to be freed.
func Threshold(shares []Signature, memberIds []int, system System) (Signature, error) {

	// Check the list length.
	if len(shares) == 0 {
		return Element{}, errors.New("bls.Recover: empty list")
	}
	if len(shares) != len(memberIds) {
		return Element{}, errors.New("bls.Recover: list length mismatch")
	}

	// Determine the group order.
	n := (C.mpz_sizeinbase(&system.pairing.get.r[0], 2) + 7) / 8
	bytes := make([]byte, n)
	C.mpz_export(unsafe.Pointer(&bytes[0]), &n, 1, 1, 1, 0, &system.pairing.get.r[0])
	r := big.NewInt(0).SetBytes(bytes)

	// Calculate sigma.
	sigma := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_G1(sigma, system.pairing.get)
	C.element_set1(sigma)
	var p *big.Int
	var q *big.Int
	u := big.NewInt(0)
	v := big.NewInt(0)
	var lambda C.mpz_t
	C.mpz_init(&lambda[0])
	s := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_G1(s, system.pairing.get)
	for i := range memberIds {

		// Calculate lambda.
		p = big.NewInt(1)
		q = big.NewInt(1)
		for j := range memberIds {
			if memberIds[i] != memberIds[j] {
				p.Mul(p, u.Neg(big.NewInt(int64(memberIds[j]+1))))
				q.Mul(q, v.Sub(big.NewInt(int64(memberIds[i]+1)), big.NewInt(int64(memberIds[j]+1))))
			}
		}
		bytes = u.Mod(u.Mul(u.Mod(p, r), v.Mod(v.ModInverse(q, r), r)), r).Bytes()
		if len(bytes) == 0 {
			C.mpz_set_si(&lambda[0], 0)
		} else {
			C.mpz_import(&lambda[0], C.size_t(len(bytes)), 1, 1, 1, 0, unsafe.Pointer(&bytes[0]))
		}

		// Update the accumulator.
		C.element_pow_mpz(s, shares[i].get, &lambda[0])
		C.element_mul(sigma, sigma, s)

	}

	// Clean up.
	C.element_clear(s)
	C.mpz_clear(&lambda[0])

	// Return the threshold signature.
	return Element{sigma}, nil

}

// SigToBytes converts a signature to a byte slice.
func (system System) SigToBytes(signature Signature) []byte {
	n := int(C.pairing_length_in_bytes_compressed_G1(system.pairing.get))
	if n < 1 {
		return nil
	}
	bytes := make([]byte, n)
	C.element_to_bytes_compressed((*C.uchar)(unsafe.Pointer(&bytes[0])), signature.get)
	return bytes
}

// SigFromBytes converts a byte slice to a signature.
func (system System) SigFromBytes(bytes []byte) (Signature, error) {
	n := int(C.pairing_length_in_bytes_compressed_G1(system.pairing.get))
	if n != len(bytes) {
		return Element{}, errors.New("bls.FromBytes: signature length mismatch")
	}
	sigma := (*C.struct_element_s)(C.malloc(sizeOfElement))
	C.element_init_G1(sigma, system.pairing.get)
	C.element_from_bytes_compressed(sigma, (*C.uchar)(unsafe.Pointer(&bytes[0])))
	return Element{sigma}, nil
}

// Free the memory occupied by the element. The element cannot be used after
// calling this function.
func (element Element) Free() {
	C.element_clear(element.get)
}

// Free the memory occupied by the pairing parameters. The parameters cannot be
// used after calling this function.
func (params Params) Free() {
	C.pbc_param_clear(params.get)
}

// Free the memory occupied by the pairing. The pairing cannot be used after
// calling this function.
func (pairing Pairing) Free() {
	C.pairing_clear(pairing.get)
}

// Free the memory occupied by the cryptosystem. The cryptosystem cannot be used
// after calling this function.
func (system System) Free() {
	system.g.Free()
}

// ToBytes exports the System to a byte slice.
func (system System) ToBytes() []byte {
	n := int(C.pairing_length_in_bytes_compressed_G2(system.pairing.get))
	if n < 1 {
		return nil
	}
	bytes := make([]byte, n)
	C.element_to_bytes_compressed((*C.uchar)(unsafe.Pointer(&bytes[0])), system.g.get)
	return bytes
}

// Free the memory occupied by the public key. The public key cannot be used
// after calling this function.
func (key PublicKey) Free() {
	key.gx.Free()
}

// Free the memory occupied by the private key. The private key cannot be used
// after calling this function.
func (secret PrivateKey) Free() {
	secret.x.Free()
}
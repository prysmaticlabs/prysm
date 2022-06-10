// Copyright 2019, 2020 Weald Technology Trading
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	bls "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/pkg/errors"
)

// BLSPublicKey used in the BLS signature scheme.
type BLSPublicKey struct {
	key *bls.PublicKey
}

// BLSPublicKeyFromBytes creates a BLS public key from a byte slice.
func BLSPublicKeyFromBytes(pub []byte) (*BLSPublicKey, error) {
	if len(pub) != 48 {
		return nil, errors.New("public key must be 48 bytes")
	}
	var key bls.PublicKey
	if err := key.Deserialize(pub); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize public key")
	}
	return &BLSPublicKey{key: &key}, nil
}

// Aggregate two public keys.  This updates the value of the existing key.
func (k *BLSPublicKey) Aggregate(other PublicKey) {
	k.key.Add(other.(*BLSPublicKey).key)
}

// Marshal a BLS public key into a byte slice.
func (k *BLSPublicKey) Marshal() []byte {
	return k.key.Serialize()
}

// Copy creates a copy of the public key.
func (k *BLSPublicKey) Copy() PublicKey {
	bytes := k.Marshal()
	var newKey bls.PublicKey
	//#nosec G104
	_ = newKey.Deserialize(bytes)
	return &BLSPublicKey{key: &newKey}
}

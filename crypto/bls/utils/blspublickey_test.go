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

package utils_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	e2types "github.com/wealdtech/go-eth2-types/v2"
)

func TestBLSPublicKeyFromBytes(t *testing.T) {
	privBytes, err := hex.DecodeString("25295f0d1d592a90b333e26e85149708208e9f8e8bc18f6c77bd62f8ad7a6866")
	require.Nil(t, err)
	priv, err := e2types.BLSPrivateKeyFromBytes(privBytes)
	require.Nil(t, err)

	goodBytes := priv.PublicKey().Marshal()
	_, err = e2types.BLSPublicKeyFromBytes(goodBytes)
	assert.Nil(t, err)

	badBytes, err := hex.DecodeString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	require.Nil(t, err)
	_, err = e2types.BLSPublicKeyFromBytes(badBytes)
	assert.NotNil(t, err)
}

func TestBLSPublicKey(t *testing.T) {
	privKey1, err := e2types.GenerateBLSPrivateKey()
	require.Nil(t, err)

	pubKey1 := privKey1.PublicKey()
	bytes := pubKey1.Marshal()

	pubKey1Copy, err := e2types.BLSPublicKeyFromBytes(bytes)
	require.Nil(t, err)

	assert.Equal(t, pubKey1.Marshal(), pubKey1Copy.Marshal())

	_, err = e2types.BLSPublicKeyFromBytes(bytes[:46])
	require.NotNil(t, err)

	privKey2, err := e2types.GenerateBLSPrivateKey()
	require.Nil(t, err)
	pubKey2 := privKey2.PublicKey()

	aggPubKey1 := pubKey1.Copy()
	aggPubKey1.Aggregate(pubKey2)
	aggPubKey2 := pubKey2.Copy()
	aggPubKey2.Aggregate(pubKey1)
	assert.Equal(t, aggPubKey1.Marshal(), aggPubKey2.Marshal())
}

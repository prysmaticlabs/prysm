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
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	e2types "github.com/wealdtech/go-eth2-types/v2"
)

func TestInvalidSignatureFromBytes(t *testing.T) {
	_, err := e2types.BLSSignatureFromBytes([]byte{0x00})
	require.NotNil(t, err)
}

func TestAggregate(t *testing.T) {
	pk1, err := e2types.GenerateBLSPrivateKey()
	require.Nil(t, err)
	pk2, err := e2types.GenerateBLSPrivateKey()
	require.Nil(t, err)
	pk3, err := e2types.GenerateBLSPrivateKey()
	require.Nil(t, err)

	msgs := make([][]byte, 3)
	msg0 := [32]byte{}
	_, err = rand.Read(msg0[:])
	require.Nil(t, err)
	msgs[0] = msg0[:]
	msg1 := [32]byte{}
	_, err = rand.Read(msg1[:])
	require.Nil(t, err)
	msgs[1] = msg1[:]
	msg2 := [32]byte{}
	_, err = rand.Read(msg2[:])
	require.Nil(t, err)
	msgs[2] = msg2[:]

	pubKeys := make([]e2types.PublicKey, 3)
	pubKeys[0] = pk1.PublicKey()
	pubKeys[1] = pk2.PublicKey()
	pubKeys[2] = pk3.PublicKey()

	sigs := make([]e2types.Signature, 3)
	sigs[0] = pk1.Sign(msgs[0])
	sigs[1] = pk2.Sign(msgs[1])
	sigs[2] = pk3.Sign(msgs[2])

	sig := e2types.AggregateSignatures(sigs)

	verified := sig.VerifyAggregate(msgs, pubKeys)
	assert.True(t, verified)
}

func TestAggregateNoSigs(t *testing.T) {
	pk1, err := e2types.GenerateBLSPrivateKey()
	require.Nil(t, err)

	pubKeys := make([]e2types.PublicKey, 0)

	msg := []byte("A test message")
	sig := pk1.Sign(msg)

	verified := sig.VerifyAggregateCommon(msg, pubKeys)
	assert.False(t, verified)

	msgs := make([][]byte, 0)
	verified = sig.VerifyAggregate(msgs, pubKeys)
	assert.False(t, verified)
}

func TestAggregateCommon(t *testing.T) {
	pk1, err := e2types.GenerateBLSPrivateKey()
	require.Nil(t, err)
	pk2, err := e2types.GenerateBLSPrivateKey()
	require.Nil(t, err)
	pk3, err := e2types.GenerateBLSPrivateKey()
	require.Nil(t, err)

	pubKeys := make([]e2types.PublicKey, 3)
	pubKeys[0] = pk1.PublicKey()
	pubKeys[1] = pk2.PublicKey()
	pubKeys[2] = pk3.PublicKey()

	msg := []byte("A test message")
	sigs := make([]e2types.Signature, 3)
	sigs[0] = pk1.Sign(msg)
	sigs[1] = pk2.Sign(msg)
	sigs[2] = pk3.Sign(msg)

	aggSig := e2types.AggregateSignatures(sigs)

	verified := aggSig.VerifyAggregateCommon(msg, pubKeys)
	assert.True(t, verified)
}

func TestAggregateNone(t *testing.T) {
	sigs := make([]e2types.Signature, 0)
	aggSig := e2types.AggregateSignatures(sigs)
	assert.Nil(t, aggSig)
}

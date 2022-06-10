// Copyright © 2019 Weald Technology Trading
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
	"errors"
	"math/big"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/crypto/bls/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	e2types "github.com/wealdtech/go-eth2-types/v2"
)

func _bigInt(input string) *big.Int {
	res, _ := new(big.Int).SetString(input, 10)
	return res
}

func TestMain(m *testing.M) {
	if err := e2types.InitBLS(); err != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestPrivateKeyFromSeedAndPath(t *testing.T) {
	tests := []struct {
		name string
		seed []byte
		path string
		err  error
		sk   *big.Int
	}{
		{
			name: "Nil",
			err:  errors.New("no path"),
		},
		{
			name: "EmptyPath",
			path: "",
			err:  errors.New("no path"),
		},
		{
			name: "EmptySeed",
			path: "m/12381/3600/0/0",
			err:  errors.New("seed must be at least 128 bits"),
		},
		{
			name: "BadPath1",
			seed: _byteArray("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"),
			path: "m/bad path",
			err:  errors.New(`invalid index "bad path" at path component 1`),
		},
		{
			name: "BadPath2",
			seed: _byteArray("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"),
			path: "m/m/12381",
			err:  errors.New(`invalid master at path component 1`),
		},
		{
			name: "BadPath3",
			seed: _byteArray("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"),
			path: "1/m/12381",
			err:  errors.New(`not master at path component 0`),
		},
		{
			name: "BadPath4",
			seed: _byteArray("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"),
			path: "m/12381//0",
			err:  errors.New(`no entry at path component 2`),
		},
		{
			name: "BadPath5",
			seed: _byteArray("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"),
			path: "m/12381/-1/0",
			err:  errors.New(`invalid index "-1" at path component 2`),
		},
		{
			name: "Good1",
			seed: _byteArray("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"),
			path: "m/12381/3600/0/0",
			sk:   _bigInt("46177761799149885423324319418907178427534014236612345059251079131808426427278"),
		},
		{
			name: "Good2",
			seed: _byteArray("52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c64981855ad8681d0d86d1e91e00167939cb6694d2c422acd208a0072939487f6999"),
			path: "m/12381/3600/0/0",
			sk:   _bigInt("42833789910372195542782452087346535004799190497837791522284717918803358261356"),
		},
		{
			name: "Spec0",
			seed: _byteArray("c55257c360c07c72029aebc1b53c05ed0362ada38ead3e3e9efa3708e53495531f09a6987599d18264c1e1c92f2cf141630c7a3c4ab7c81b2f001698e7463b04"),
			path: "m/0",
			sk:   _bigInt("20397789859736650942317412262472558107875392172444076792671091975210932703118"),
		},
		{
			name: "Spec1",
			seed: _byteArray("3141592653589793238462643383279502884197169399375105820974944592"),
			path: "m/3141592653",
			sk:   _bigInt("25457201688850691947727629385191704516744796114925897962676248250929345014287"),
		},
		{
			name: "Spec2",
			seed: _byteArray("0099FF991111002299DD7744EE3355BBDD8844115566CC55663355668888CC00"),
			path: "m/4294967295",
			sk:   _bigInt("29358610794459428860402234341874281240803786294062035874021252734817515685787"),
		},
		{
			name: "Spec3",
			seed: _byteArray("d4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3"),
			path: "m/42",
			sk:   _bigInt("31372231650479070279774297061823572166496564838472787488249775572789064611981"),
		},
		{
			name: "IndexTooBig",
			seed: _byteArray("0099FF991111002299DD7744EE3355BBDD8844115566CC55663355668888CC00"),
			path: "m/4294967296",
			err:  errors.New(`invalid index "4294967296" at path component 1`),
		},
		{
			name: "IndexNegative",
			seed: _byteArray("0099FF991111002299DD7744EE3355BBDD8844115566CC55663355668888CC00"),
			path: "m/-1",
			err:  errors.New(`invalid index "-1" at path component 1`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sk, err := utils.PrivateKeyFromSeedAndPath(test.seed, test.path)
			if test.err != nil {
				require.NotNil(t, err)
				assert.Equal(t, test.err.Error(), err.Error())
			} else {
				require.Nil(t, err)
				// fmt.Printf("%v\n", new(big.Int).SetBytes(sk.Marshal()))
				assert.Equal(t, test.sk.Bytes(), sk.Marshal())
			}
		})
	}
}

func TestShortPrivateKey(t *testing.T) {
	seed := _byteArray("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	path := "m/12381/3600/0/41"
	sk, err := utils.PrivateKeyFromSeedAndPath(seed, path)
	assert.Nil(t, err)
	assert.Equal(t, _bigInt("40053195758832663164718180086452958519214934897695771517699548485069286510185").Bytes(), sk.Marshal())
}

func TestDeriveMasterKey(t *testing.T) {
	tests := []struct {
		name string
		seed []byte
		err  error
		sk   *big.Int
	}{
		{
			name: "ShortSeed",
			seed: _byteArray("0102030405060708090a0b0c0d0e"),
			err:  errors.New("seed must be at least 128 bits"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sk, err := utils.DeriveMasterSK(test.seed)
			if test.err != nil {
				require.NotNil(t, err)
				assert.Equal(t, test.err.Error(), err.Error())
			} else {
				require.Nil(t, err)
				assert.Equal(t, test.sk, sk)
			}
		})
	}
}

func TestDeriveChildKey(t *testing.T) {
	tests := []struct {
		name       string
		seed       []byte
		childIndex uint32
		err        error
		childSK    *big.Int
	}{
		// TODO
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			masterSK, err := utils.DeriveMasterSK(test.seed)
			require.Nil(t, err)
			childSK, err := utils.DeriveChildSK(masterSK, test.childIndex)
			if test.err != nil {
				require.NotNil(t, err)
				require.Equal(t, test.err.Error(), err.Error())
			} else {
				require.Nil(t, err)
				assert.Equal(t, test.childSK, childSK)
			}
		})
	}
}

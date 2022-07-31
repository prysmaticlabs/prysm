package blocks

import (
	"bytes"
	"testing"

	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func Test_NewSignedBeaconBlock(t *testing.T) {
	t.Run("GenericSignedBeaconBlock_Phase0", func(t *testing.T) {
		pb := &eth.GenericSignedBeaconBlock_Phase0{
			Phase0: &eth.SignedBeaconBlock{
				Block: &eth.BeaconBlock{
					Body: &eth.BeaconBlockBody{}}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Phase0, b.version)
	})
	t.Run("SignedBeaconBlock", func(t *testing.T) {
		pb := &eth.SignedBeaconBlock{
			Block: &eth.BeaconBlock{
				Body: &eth.BeaconBlockBody{}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Phase0, b.version)
	})
	t.Run("GenericSignedBeaconBlock_Altair", func(t *testing.T) {
		pb := &eth.GenericSignedBeaconBlock_Altair{
			Altair: &eth.SignedBeaconBlockAltair{
				Block: &eth.BeaconBlockAltair{
					Body: &eth.BeaconBlockBodyAltair{}}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Altair, b.version)
	})
	t.Run("SignedBeaconBlockAltair", func(t *testing.T) {
		pb := &eth.SignedBeaconBlockAltair{
			Block: &eth.BeaconBlockAltair{
				Body: &eth.BeaconBlockBodyAltair{}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Altair, b.version)
	})
	t.Run("GenericSignedBeaconBlock_Bellatrix", func(t *testing.T) {
		pb := &eth.GenericSignedBeaconBlock_Bellatrix{
			Bellatrix: &eth.SignedBeaconBlockBellatrix{
				Block: &eth.BeaconBlockBellatrix{
					Body: &eth.BeaconBlockBodyBellatrix{}}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Bellatrix, b.version)
	})
	t.Run("SignedBeaconBlockBellatrix", func(t *testing.T) {
		pb := &eth.SignedBeaconBlockBellatrix{
			Block: &eth.BeaconBlockBellatrix{
				Body: &eth.BeaconBlockBodyBellatrix{}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Bellatrix, b.version)
	})
	t.Run("GenericSignedBeaconBlock_BlindedBellatrix", func(t *testing.T) {
		pb := &eth.GenericSignedBeaconBlock_BlindedBellatrix{
			BlindedBellatrix: &eth.SignedBlindedBeaconBlockBellatrix{
				Block: &eth.BlindedBeaconBlockBellatrix{
					Body: &eth.BlindedBeaconBlockBodyBellatrix{}}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.BellatrixBlind, b.version)
	})
	t.Run("SignedBlindedBeaconBlockBellatrix", func(t *testing.T) {
		pb := &eth.SignedBlindedBeaconBlockBellatrix{
			Block: &eth.BlindedBeaconBlockBellatrix{
				Body: &eth.BlindedBeaconBlockBodyBellatrix{}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.BellatrixBlind, b.version)
	})
	t.Run("nil", func(t *testing.T) {
		_, err := NewSignedBeaconBlock(nil)
		assert.ErrorContains(t, "attempted to wrap nil object", err)
	})
	t.Run("unsupported type", func(t *testing.T) {
		_, err := NewSignedBeaconBlock(&bytes.Reader{})
		assert.ErrorContains(t, "unable to create block from type *bytes.Reader", err)
	})
}

func Test_NewBeaconBlock(t *testing.T) {
	t.Run("GenericBeaconBlock_Phase0", func(t *testing.T) {
		pb := &eth.GenericBeaconBlock_Phase0{Phase0: &eth.BeaconBlock{Body: &eth.BeaconBlockBody{}}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Phase0, b.version)
	})
	t.Run("BeaconBlock", func(t *testing.T) {
		pb := &eth.BeaconBlock{Body: &eth.BeaconBlockBody{}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Phase0, b.version)
	})
	t.Run("GenericBeaconBlock_Altair", func(t *testing.T) {
		pb := &eth.GenericBeaconBlock_Altair{Altair: &eth.BeaconBlockAltair{Body: &eth.BeaconBlockBodyAltair{}}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Altair, b.version)
	})
	t.Run("BeaconBlockAltair", func(t *testing.T) {
		pb := &eth.BeaconBlockAltair{Body: &eth.BeaconBlockBodyAltair{}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Altair, b.version)
	})
	t.Run("GenericBeaconBlock_Bellatrix", func(t *testing.T) {
		pb := &eth.GenericBeaconBlock_Bellatrix{Bellatrix: &eth.BeaconBlockBellatrix{Body: &eth.BeaconBlockBodyBellatrix{}}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Bellatrix, b.version)
	})
	t.Run("BeaconBlockBellatrix", func(t *testing.T) {
		pb := &eth.BeaconBlockBellatrix{Body: &eth.BeaconBlockBodyBellatrix{}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Bellatrix, b.version)
	})
	t.Run("GenericBeaconBlock_BlindedBellatrix", func(t *testing.T) {
		pb := &eth.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: &eth.BlindedBeaconBlockBellatrix{Body: &eth.BlindedBeaconBlockBodyBellatrix{}}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.BellatrixBlind, b.version)
	})
	t.Run("BlindedBeaconBlockBellatrix", func(t *testing.T) {
		pb := &eth.BlindedBeaconBlockBellatrix{Body: &eth.BlindedBeaconBlockBodyBellatrix{}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.BellatrixBlind, b.version)
	})
	t.Run("nil", func(t *testing.T) {
		_, err := NewBeaconBlock(nil)
		assert.ErrorContains(t, "attempted to wrap nil object", err)
	})
	t.Run("unsupported type", func(t *testing.T) {
		_, err := NewBeaconBlock(&bytes.Reader{})
		assert.ErrorContains(t, "unable to create block from type *bytes.Reader", err)
	})
}

func Test_NewBeaconBlockBody(t *testing.T) {
	t.Run("BeaconBlockBody", func(t *testing.T) {
		pb := &eth.BeaconBlockBody{}
		b, err := NewBeaconBlockBody(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Phase0, b.version)
	})
	t.Run("BeaconBlockBodyAltair", func(t *testing.T) {
		pb := &eth.BeaconBlockBodyAltair{}
		b, err := NewBeaconBlockBody(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Altair, b.version)
	})
	t.Run("BeaconBlockBodyBellatrix", func(t *testing.T) {
		pb := &eth.BeaconBlockBodyBellatrix{}
		b, err := NewBeaconBlockBody(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Bellatrix, b.version)
	})
	t.Run("BlindedBeaconBlockBodyBellatrix", func(t *testing.T) {
		pb := &eth.BlindedBeaconBlockBodyBellatrix{}
		b, err := NewBeaconBlockBody(pb)
		require.NoError(t, err)
		assert.Equal(t, version.BellatrixBlind, b.version)
	})
	t.Run("nil", func(t *testing.T) {
		_, err := NewBeaconBlockBody(nil)
		assert.ErrorContains(t, "attempted to wrap nil object", err)
	})
	t.Run("unsupported type", func(t *testing.T) {
		_, err := NewBeaconBlockBody(&bytes.Reader{})
		assert.ErrorContains(t, "unable to create block body from type *bytes.Reader", err)
	})
}

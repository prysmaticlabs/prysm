package apimiddleware

import (
	"encoding/base64"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestUnmarshalEpochParticipation(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		b := []byte{3, 3, 0}
		b64 := []byte("\"" + base64.StdEncoding.EncodeToString(b) + "\"")
		ep := EpochParticipation{}
		require.NoError(t, ep.UnmarshalJSON(b64))
		require.Equal(t, 3, len(ep))
		assert.Equal(t, "3", ep[0])
		assert.Equal(t, "3", ep[1])
		assert.Equal(t, "0", ep[2])
	})
	t.Run("incorrect value", func(t *testing.T) {
		ep := EpochParticipation{}
		err := ep.UnmarshalJSON([]byte(":illegal:"))
		require.NotNil(t, err)
		assert.ErrorContains(t, "provided epoch participation json string is malformed", err)
	})
	t.Run("length too small", func(t *testing.T) {
		ep := EpochParticipation{}
		err := ep.UnmarshalJSON([]byte("x"))
		require.NotNil(t, err)
		assert.ErrorContains(t, "epoch participation length must be at least 2", err)
	})
	t.Run("null value", func(t *testing.T) {
		ep := EpochParticipation{}
		require.NoError(t, ep.UnmarshalJSON([]byte("null")))
		assert.DeepEqual(t, EpochParticipation([]string{}), ep)
	})
	t.Run("invalid value", func(t *testing.T) {
		ep := EpochParticipation{}
		require.ErrorContains(t, "provided epoch participation json string is malformed", ep.UnmarshalJSON([]byte("XdHJ1ZQ==X")))
	})
}

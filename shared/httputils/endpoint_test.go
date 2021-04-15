package httputils

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/httputils/authorizationmethod"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestToHeaderValue(t *testing.T) {
	t.Run("None", func(t *testing.T) {
		data := &AuthorizationData{
			Method: authorizationmethod.None,
			Value:  "foo",
		}
		header, err := data.ToHeaderValue()
		require.NoError(t, err)
		assert.Equal(t, "", header)
	})
	t.Run("Basic", func(t *testing.T) {
		data := &AuthorizationData{
			Method: authorizationmethod.Basic,
			Value:  "foo",
		}
		header, err := data.ToHeaderValue()
		require.NoError(t, err)
		assert.Equal(t, "Basic foo", header)
	})
	t.Run("Bearer", func(t *testing.T) {
		data := &AuthorizationData{
			Method: authorizationmethod.Bearer,
			Value:  "foo",
		}
		header, err := data.ToHeaderValue()
		require.NoError(t, err)
		assert.Equal(t, "Bearer foo", header)
	})
	t.Run("Unknown", func(t *testing.T) {
		data := &AuthorizationData{
			Method: 99,
			Value:  "foo",
		}
		_, err := data.ToHeaderValue()
		require.NotNil(t, err)
	})
}

func TestMethod(t *testing.T) {
	t.Run("None", func(t *testing.T) {
		method := Method("")
		assert.Equal(t, authorizationmethod.None, method)
		method = Method("foo")
		assert.Equal(t, authorizationmethod.None, method)
	})
	t.Run("Basic", func(t *testing.T) {
		method := Method("Basic")
		assert.Equal(t, authorizationmethod.Basic, method)
	})
	t.Run("Basic different text case", func(t *testing.T) {
		method := Method("bAsIc")
		assert.Equal(t, authorizationmethod.Basic, method)
	})
	t.Run("Bearer", func(t *testing.T) {
		method := Method("Bearer")
		assert.Equal(t, authorizationmethod.Bearer, method)
	})
	t.Run("Bearer different text case", func(t *testing.T) {
		method := Method("bEaReR")
		assert.Equal(t, authorizationmethod.Bearer, method)
	})
}

func TestEndpointEquals(t *testing.T) {
	e := Endpoint{
		Url: "Url",
		Auth: AuthorizationData{
			Method: authorizationmethod.Basic,
			Value:  "Basic username:password",
		},
	}

	t.Run("equal", func(t *testing.T) {
		other := Endpoint{
			Url: "Url",
			Auth: AuthorizationData{
				Method: authorizationmethod.Basic,
				Value:  "Basic username:password",
			},
		}
		assert.Equal(t, true, e.Equals(other))
	})
	t.Run("different URL", func(t *testing.T) {
		other := Endpoint{
			Url: "Different",
			Auth: AuthorizationData{
				Method: authorizationmethod.Basic,
				Value:  "Basic username:password",
			},
		}
		assert.Equal(t, false, e.Equals(other))
	})
	t.Run("different auth data", func(t *testing.T) {
		other := Endpoint{
			Url: "Url",
			Auth: AuthorizationData{
				Method: authorizationmethod.Bearer,
				Value:  "Bearer token",
			},
		}
		assert.Equal(t, false, e.Equals(other))
	})
}

func TestAuthorizationDataEquals(t *testing.T) {
	d := AuthorizationData{
		Method: authorizationmethod.Basic,
		Value:  "username:password",
	}

	t.Run("equal", func(t *testing.T) {
		other := AuthorizationData{
			Method: authorizationmethod.Basic,
			Value:  "username:password",
		}
		assert.Equal(t, true, d.Equals(other))
	})
	t.Run("different method", func(t *testing.T) {
		other := AuthorizationData{
			Method: authorizationmethod.None,
			Value:  "username:password",
		}
		assert.Equal(t, false, d.Equals(other))
	})
	t.Run("different value", func(t *testing.T) {
		other := AuthorizationData{
			Method: authorizationmethod.Basic,
			Value:  "different:different",
		}
		assert.Equal(t, false, d.Equals(other))
	})
}

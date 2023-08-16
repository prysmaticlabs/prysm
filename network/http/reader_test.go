package http

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestSSZRequested(t *testing.T) {
	t.Run("ssz_requested", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{octetStreamMediaType}
		result, err := SszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("ssz_content_type_first", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{fmt.Sprintf("%s,%s", octetStreamMediaType, jsonMediaType)}
		result, err := SszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("ssz_content_type_preferred_1", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{fmt.Sprintf("%s;q=0.9,%s", jsonMediaType, octetStreamMediaType)}
		result, err := SszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("ssz_content_type_preferred_2", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{fmt.Sprintf("%s;q=0.95,%s;q=0.9", octetStreamMediaType, jsonMediaType)}
		result, err := SszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("other_content_type_preferred", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{fmt.Sprintf("%s,%s;q=0.9", jsonMediaType, octetStreamMediaType)}
		result, err := SszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("other_params", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{fmt.Sprintf("%s,%s;q=0.9,otherparam=xyz", jsonMediaType, octetStreamMediaType)}
		result, err := SszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("no_header", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		result, err := SszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("empty_header", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{}
		result, err := SszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("empty_header_value", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{""}
		result, err := SszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("other_content_type", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{"application/other"}
		result, err := SszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("garbage", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example", nil)
		request.Header["Accept"] = []string{"This is Sparta!!!"}
		result, err := SszRequested(request)
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})
}

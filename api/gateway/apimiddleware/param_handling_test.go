package apimiddleware

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestHandleURLParameters(t *testing.T) {
	var body bytes.Buffer

	t.Run("no_params", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example/bar", &body)

		errJson := HandleURLParameters("/not_param", request, []string{})
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, "/bar", request.URL.Path)
	})

	t.Run("with_params", func(t *testing.T) {
		muxVars := make(map[string]string)
		muxVars["bar_param"] = "bar"
		muxVars["quux_param"] = "quux"
		request := httptest.NewRequest("GET", "http://foo.example/bar/baz/quux", &body)
		request = mux.SetURLVars(request, muxVars)

		errJson := HandleURLParameters("/{bar_param}/not_param/{quux_param}", request, []string{})
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, "/YmFy/baz/cXV1eA==", request.URL.Path)
	})

	t.Run("with_literal", func(t *testing.T) {
		muxVars := make(map[string]string)
		muxVars["bar_param"] = "bar"
		request := httptest.NewRequest("GET", "http://foo.example/bar/baz", &body)
		request = mux.SetURLVars(request, muxVars)

		errJson := HandleURLParameters("/{bar_param}/not_param/", request, []string{"bar_param"})
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, "/bar/baz", request.URL.Path)
	})

	t.Run("with_hex", func(t *testing.T) {
		muxVars := make(map[string]string)
		muxVars["hex_param"] = "0x626172"
		request := httptest.NewRequest("GET", "http://foo.example/0x626172/baz", &body)
		request = mux.SetURLVars(request, muxVars)

		errJson := HandleURLParameters("/{hex_param}/not_param/", request, []string{})
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, "/YmFy/baz", request.URL.Path)
	})
}

func TestHandleQueryParameters(t *testing.T) {
	var body bytes.Buffer

	t.Run("regular_params", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example?bar=bar&baz=baz", &body)

		errJson := HandleQueryParameters(request, []QueryParam{{Name: "bar"}, {Name: "baz"}})
		require.Equal(t, true, errJson == nil)
		query := request.URL.Query()
		v, ok := query["bar"]
		require.Equal(t, true, ok, "query param not found")
		require.Equal(t, 1, len(v), "wrong number of query param values")
		assert.Equal(t, "bar", v[0])
		v, ok = query["baz"]
		require.Equal(t, true, ok, "query param not found")
		require.Equal(t, 1, len(v), "wrong number of query param values")
		assert.Equal(t, "baz", v[0])
	})

	t.Run("hex_and_enum_params", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://foo.example?hex=0x626172&baz=baz", &body)

		errJson := HandleQueryParameters(request, []QueryParam{{Name: "hex", Hex: true}, {Name: "baz", Enum: true}})
		require.Equal(t, true, errJson == nil)
		query := request.URL.Query()
		v, ok := query["hex"]
		require.Equal(t, true, ok, "query param not found")
		require.Equal(t, 1, len(v), "wrong number of query param values")
		assert.Equal(t, "YmFy", v[0])
		v, ok = query["baz"]
		require.Equal(t, true, ok, "query param not found")
		require.Equal(t, 1, len(v), "wrong number of query param values")
		assert.Equal(t, "BAZ", v[0])
	})
}

func TestIsRequestParam(t *testing.T) {
	tests := []struct {
		s string
		b bool
	}{
		{"", false},
		{"{", false},
		{"}", false},
		{"{}", false},
		{"{x}", true},
		{"{very_long_parameter_name_with_underscores}", true},
	}
	for _, tt := range tests {
		b := isRequestParam(tt.s)
		assert.Equal(t, tt.b, b)
	}
}

func TestNormalizeQueryValues(t *testing.T) {
	input := make(map[string][]string)
	input["key"] = []string{"value1", "value2,value3,value4", "value5"}

	normalizeQueryValues(input)
	require.Equal(t, 5, len(input["key"]))
	assert.Equal(t, "value1", input["key"][0])
	assert.Equal(t, "value2", input["key"][1])
	assert.Equal(t, "value3", input["key"][2])
	assert.Equal(t, "value4", input["key"][3])
	assert.Equal(t, "value5", input["key"][4])
}

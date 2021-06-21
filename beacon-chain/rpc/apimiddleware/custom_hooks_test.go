package apimiddleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/gateway"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestWrapAttestationArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitAttestationRequestJson{},
		}
		unwrappedAtts := []*attestationJson{{AggregationBits: "1010"}}
		unwrappedAttsJson, err := json.Marshal(unwrappedAtts)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedAttsJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapAttestationsArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		wrappedAtts := &submitAttestationRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedAtts))
		require.Equal(t, 1, len(wrappedAtts.Data), "wrong number of wrapped attestations")
		assert.Equal(t, "1010", wrappedAtts.Data[0].AggregationBits)
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitAttestationRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapAttestationsArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode attestations array"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestPrepareGraffiti(t *testing.T) {
	endpoint := gateway.Endpoint{
		PostRequest: &beaconBlockContainerJson{
			Message: &beaconBlockJson{
				Body: &beaconBlockBodyJson{},
			},
		},
	}

	t.Run("32_bytes", func(t *testing.T) {
		endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti = string(bytesutil.PadTo([]byte("foo"), 32))

		prepareGraffiti(endpoint, nil, nil)
		assert.Equal(
			t,
			"0x666f6f0000000000000000000000000000000000000000000000000000000000",
			endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti,
		)
	})

	t.Run("less_than_32_bytes", func(t *testing.T) {
		endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti = "foo"

		prepareGraffiti(endpoint, nil, nil)
		assert.Equal(
			t,
			"0x666f6f0000000000000000000000000000000000000000000000000000000000",
			endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti,
		)
	})

	t.Run("more_than_32_bytes", func(t *testing.T) {
		endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti = string(bytesutil.PadTo([]byte("foo"), 33))

		prepareGraffiti(endpoint, nil, nil)
		assert.Equal(
			t,
			"0x666f6f0000000000000000000000000000000000000000000000000000000000",
			endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti,
		)
	})
}

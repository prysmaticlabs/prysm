package shared

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/lookup"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
)

func TestDecodeError(t *testing.T) {
	e := errors.New("not a number")
	de := NewDecodeError(e, "Z")
	de = NewDecodeError(de, "Y")
	de = NewDecodeError(de, "X")
	assert.Equal(t, "could not decode X.Y.Z: not a number", de.Error())
}

// TestWriteStateFetchError tests the WriteStateFetchError function
// to ensure that the correct error message and code are written to the response
// as an expected JSON format.
func TestWriteStateFetchError(t *testing.T) {
	cases := []struct {
		err             error
		expectedMessage string
		expectedCode    int
	}{
		{
			err:             &lookup.StateNotFoundError{},
			expectedMessage: "State not found",
			expectedCode:    http.StatusNotFound,
		},
		{
			err:             &lookup.StateIdParseError{},
			expectedMessage: "Invalid state ID",
			expectedCode:    http.StatusBadRequest,
		},
		{
			err:             errors.New("state not found"),
			expectedMessage: "Could not get state",
			expectedCode:    http.StatusInternalServerError,
		},
	}

	for _, c := range cases {
		writer := httptest.NewRecorder()
		WriteStateFetchError(writer, c.err)

		assert.Equal(t, c.expectedCode, writer.Code, "incorrect status code")
		assert.StringContains(t, c.expectedMessage, writer.Body.String(), "incorrect error message")

		e := &http2.DefaultErrorJson{}
		assert.NoError(t, json.Unmarshal(writer.Body.Bytes(), e), "failed to unmarshal response")
	}
}

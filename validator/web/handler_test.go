package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestWebHandler(t *testing.T) {
	tests := []struct{
		name string
		requestURI string
		wantStatus int
	}{
		{
			name: "base route",
			requestURI: "/",
			wantStatus: 200,
		},
		{
			name: "index.html",
			requestURI: "/index.html",
			wantStatus: 200,
		},
		{
			name: "bad route",
			requestURI: "/foobar_bad",
			wantStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{RequestURI: tt.requestURI}
			res := httptest.NewRecorder()
			webHandler(res, req)
			assert.Equal(t, tt.wantStatus, res.Result().StatusCode)
		})
	}
}

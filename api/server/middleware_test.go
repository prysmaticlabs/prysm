package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeQueryValuesHandler(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("next handler"))
	})

	handler := NormalizeQueryValuesHandler(nextHandler)

	tests := []struct {
		name          string
		inputQuery    string
		expectedQuery string
	}{
		{
			name:          "3 values",
			inputQuery:    "key=value1,value2,value3",
			expectedQuery: "key=value1&key=value2&key=value3", // replace with expected normalized value
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/test?"+test.inputQuery, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
			}

			if req.URL.RawQuery != test.expectedQuery {
				t.Errorf("query not normalized: got %v want %v", req.URL.RawQuery, test.expectedQuery)
			}

			if rr.Body.String() != "next handler" {
				t.Errorf("next handler was not executed")
			}
		})
	}
}

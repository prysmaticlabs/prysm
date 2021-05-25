package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

var hook *Webhook

func TestMain(m *testing.M) {
	var err error
	hook, err = New(Options.Secret("Foobar!"))
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func newServer(handler http.HandlerFunc) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc(path, handler)
	return httptest.NewServer(mux)
}

func TestBadRequests(t *testing.T) {
	tests := []struct {
		name    string
		event   Event
		payload io.Reader
		headers http.Header
	}{
		{
			name:    "BadNoEventHeader",
			event:   CreateEvent,
			payload: bytes.NewBuffer([]byte("{}")),
			headers: http.Header{},
		},
		{
			name:    "UnsubscribedEvent",
			event:   CreateEvent,
			payload: bytes.NewBuffer([]byte("{}")),
			headers: http.Header{
				"X-Github-Event": []string{"noneexistant_event"},
			},
		},
		{
			name:    "BadBody",
			event:   CommitCommentEvent,
			payload: bytes.NewBuffer([]byte("")),
			headers: http.Header{
				"X-Github-Event":  []string{"commit_comment"},
				"X-Hub-Signature": []string{"sha1=156404ad5f721c53151147f3d3d302329f95a3ab"},
			},
		},
		{
			name:    "BadSignatureLength",
			event:   CommitCommentEvent,
			payload: bytes.NewBuffer([]byte("{}")),
			headers: http.Header{
				"X-Github-Event":  []string{"commit_comment"},
				"X-Hub-Signature": []string{""},
			},
		},
		{
			name:    "BadSignatureMatch",
			event:   CommitCommentEvent,
			payload: bytes.NewBuffer([]byte("{}")),
			headers: http.Header{
				"X-Github-Event":  []string{"commit_comment"},
				"X-Hub-Signature": []string{"sha1=111"},
			},
		},
	}

	for _, tt := range tests {
		tc := tt
		client := &http.Client{}
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var parseError error
			server := newServer(func(w http.ResponseWriter, r *http.Request) {
				_, parseError = hook.Parse(r, tc.event)
			})
			defer server.Close()
			req, err := http.NewRequest(http.MethodPost, server.URL+path, tc.payload)
			require.NoError(t, err)
			req.Header = tc.headers
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.NotNil(t, parseError)
		})
	}
}

func TestWebhooks(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		typ      interface{}
		filename string
		headers  http.Header
	}{
		{
			name:     "PullRequestEvent",
			event:    PullRequestEvent,
			typ:      PullRequestPayload{},
			filename: "../testdata/github/pull-request.json",
			headers: http.Header{
				"X-Github-Event":  []string{"pull_request"},
				"X-Hub-Signature": []string{"sha1=88972f972db301178aa13dafaf112d26416a15e6"},
			},
		},
		{
			name:     "ReleaseEvent",
			event:    ReleaseEvent,
			typ:      ReleasePayload{},
			filename: "../testdata/github/release.json",
			headers: http.Header{
				"X-Github-Event":  []string{"release"},
				"X-Hub-Signature": []string{"sha1=e62bb4c51bc7dde195b9525971c2e3aecb394390"},
			},
		},
	}

	for _, tt := range tests {
		tc := tt
		client := &http.Client{}
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			payload, err := os.Open(tc.filename)
			require.NoError(t, err)
			defer func() {
				_ = payload.Close()
			}()

			var parseError error
			var results interface{}
			server := newServer(func(w http.ResponseWriter, r *http.Request) {
				results, parseError = hook.Parse(r, tc.event)
			})
			defer server.Close()
			req, err := http.NewRequest(http.MethodPost, server.URL+path, payload)
			require.NoError(t, err)
			req.Header = tc.headers
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.NoError(t, parseError)
			require.Equal(t, reflect.TypeOf(tc.typ), reflect.TypeOf(results))
		})
	}
}

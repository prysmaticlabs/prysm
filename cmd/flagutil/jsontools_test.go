package flagutil

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
)

type test struct {
	Foo string `json:"foo"`
	Bar int    `json:"bar"`
}

func TestUnmarshalFromFileOrURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprintf(w, `{ "foo": "foo", "bar": 1}`)
		require.NoError(t, err)
	}))
	defer srv.Close()
	ctx := context.Background()
	type args struct {
		FileOrURL string
		To        interface{}
	}
	tests := []struct {
		name        string
		args        args
		want        interface{}
		urlResponse string
		wantErr     bool
	}{
		{
			name: "Happy Path File",
			args: args{
				FileOrURL: "./testassets/test-good.json",
				To:        &test{},
			},
			want: &test{
				Foo: "foo",
				Bar: 1,
			},
			wantErr: false,
		},
		{
			name: "Happy Path URL",
			args: args{
				FileOrURL: srv.URL,
				To:        &test{},
			},
			want: &test{
				Foo: "foo",
				Bar: 1,
			},
			wantErr: false,
		},
		{
			name: "Bad File Path, not json",
			args: args{
				FileOrURL: "./jsontools.go",
				To:        &test{},
			},
			want:    &test{},
			wantErr: true,
		},
		{
			name: "Bad File Path",
			args: args{
				FileOrURL: "./test-bad.json",
				To:        &test{},
			},
			want:    &test{},
			wantErr: true,
		},
		{
			name: "Bad File Path, not found",
			args: args{
				FileOrURL: "./test-notfound.json",
				To:        &test{},
			},
			want:    &test{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UnmarshalFromFileOrURL(ctx, tt.args.FileOrURL, tt.args.To); (err != nil) != tt.wantErr {
				t.Errorf(" error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.DeepEqual(t, tt.want, tt.args.To)
		})
	}
}

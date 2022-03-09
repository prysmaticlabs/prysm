package flagutil

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
)

type test struct {
	Foo string `json:"foo"`
	Bar int    `json:"bar"`
}

func TestUnmarshalFromFileOrURL(t *testing.T) {
	ts := &test{}
	ctx := context.Background()
	type args struct {
		FileOrURL string
		To        interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "Happy Path File",
			args: args{
				FileOrURL: "./testassets/test.json",
				To:        ts,
			},
			want: &test{
				Foo: "foo",
				Bar: 1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UnmarshalFromFileOrURL(ctx, tt.args.FileOrURL, tt.args.To); (err != nil) != tt.wantErr {
				t.Errorf(" error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.DeepEqual(t, tt.args.To, tt.want)
		})
	}
}

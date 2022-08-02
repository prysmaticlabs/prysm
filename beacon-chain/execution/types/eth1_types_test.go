package types

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
)

func Test_headerToHeaderInfo(t *testing.T) {
	type args struct {
		hdr *gethTypes.Header
	}
	tests := []struct {
		name    string
		args    args
		want    *HeaderInfo
		wantErr bool
	}{
		{
			name: "OK",
			args: args{hdr: &gethTypes.Header{
				Number: big.NewInt(500),
				Time:   2345,
			}},
			want: &HeaderInfo{
				Number: big.NewInt(500),
				Hash:   common.Hash{239, 10, 13, 71, 156, 192, 23, 93, 73, 154, 255, 209, 163, 204, 129, 12, 179, 183, 65, 70, 205, 200, 57, 12, 17, 211, 209, 4, 104, 133, 73, 86},
				Time:   2345,
			},
		},
		{
			name: "nil number",
			args: args{hdr: &gethTypes.Header{
				Time: 2345,
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HeaderToHeaderInfo(tt.args.hdr)
			if (err != nil) != tt.wantErr {
				t.Errorf("headerToHeaderInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("headerToHeaderInfo() got = %v, want %v", got, tt.want)
			}
		})
	}
}

package p2putils

import (
	"reflect"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestFork(t *testing.T) {
	type args struct {
		targetEpoch types.Epoch
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.Fork
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Fork(tt.args.targetEpoch)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fork() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Fork() got = %v, want %v", got, tt.want)
			}
		})
	}
}

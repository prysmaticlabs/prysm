package p2p

import (
	"reflect"
	"testing"

	pb "github.com/ethereum/go-ethereum/sharding/p2p/proto"
)

type testStruct struct{}

func TestReverseMapping(t *testing.T) {
	tests := []struct {
		input map[pb.Topic]reflect.Type
		want  map[reflect.Type]pb.Topic
	}{
		{
			input: map[pb.Topic]reflect.Type{
				pb.Topic_UNKNOWN: reflect.TypeOf(testStruct{}),
			},
			want: map[reflect.Type]pb.Topic{
				reflect.TypeOf(testStruct{}): pb.Topic_UNKNOWN,
			},
		},
	}

	for _, tt := range tests {
		got := reverseMapping(tt.input)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("reverseMapping(%+v) = %+v. Wanted %+v", tt.input, got, tt.want)
		}
	}
}
